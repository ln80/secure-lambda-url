import * as cdk from "aws-cdk-lib";
import * as cloudfront from "aws-cdk-lib/aws-cloudfront";
import * as origins from "aws-cdk-lib/aws-cloudfront-origins";

import * as secretsmanager from "aws-cdk-lib/aws-secretsmanager";

import { Construct } from "constructs";

import * as lambda from "aws-cdk-lib/aws-lambda";
import { NodejsFunction, OutputFormat } from "aws-cdk-lib/aws-lambda-nodejs";

import * as sam from "aws-cdk-lib/aws-sam";

export class ExampleStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    const SECRET_CUSTOM_HEADER = "X-Sec-Api-Key";
    const SECRETS_MANAGER_ENDPOINT = `https://secretsmanager.${
      cdk.Stack.of(this).region
    }.amazonaws.com`;

    // Create a secret in Secrets Manager
    const secret = new secretsmanager.Secret(this, "Secret", {
      secretName: "SecretHeaderKey",
    });

    // SAM nested stack to provision a lambda extension authorizer
    const extension = new sam.CfnApplication(this, "SecureLambdaExtension", {
      location: {
        applicationId:
          "arn:aws:serverlessrepo:eu-west-1:015397314665:applications/secure-lambda-url-extension",
        semanticVersion: "0.9.0",
      },
      parameters: {},
    });

    // Define the lambda, add the extension layer,
    // and make sure to pass the required env vars
    const secureLambda = new NodejsFunction(this, "SecureLambda", {
      runtime: lambda.Runtime.NODEJS_18_X,
      entry: "src/index.ts",
      environment: {
        SECURE_LAMBDA_URL_SECRET_ENDPOINT: SECRETS_MANAGER_ENDPOINT,
        SECRETS_MANAGER_ENDPOINT: SECRETS_MANAGER_ENDPOINT,
        SECURE_LAMBDA_URL_SECRET_ARN: secret.secretArn,
        SECURE_LAMBDA_URL_HEADER_NAME: SECRET_CUSTOM_HEADER,
      },
      layers: [
        lambda.LayerVersion.fromLayerVersionArn(
          this,
          "lambdaExtension",
          extension.getAtt("Outputs.LambdaExtensionLayer").toString()
        ),
      ],
      bundling: {
        format: OutputFormat.ESM,
      },
    });

    // Grant Lambda read access to the secret
    secret.grantRead(secureLambda);

    // Get lambda function URL
    const lambdaUrl = secureLambda.addFunctionUrl({
      authType: lambda.FunctionUrlAuthType.NONE,
    });

    // Extract lambda domain name from URL
    const lambdaDomainName = cdk.Fn.select(2, cdk.Fn.split("/", lambdaUrl.url));

    // Create a CloudFront distribution
    const distribution = new cloudfront.Distribution(this, "Distribution", {
      defaultBehavior: {
        // Define Lambda Function URL as a Cloudfront HTTP Origin
        origin: new origins.HttpOrigin(lambdaDomainName, {
          customHeaders: {
            [SECRET_CUSTOM_HEADER]: cdk.Token.asString(
              secret.secretValue.unsafeUnwrap()
            ),
          },
        }),
        allowedMethods: cloudfront.AllowedMethods.ALLOW_ALL,
      },
    });

    // Use the nested SAM stack to provision a secret rotation lambda
    // that also, updates the Distribution custom header value.
    const rotation = new sam.CfnApplication(this, "RotationLambda", {
      location: {
        applicationId:
          "arn:aws:serverlessrepo:eu-west-1:015397314665:applications/secure-lambda-url-rotation",
        semanticVersion: "0.4.1",
      },

      parameters: {
        Endpoint: SECRETS_MANAGER_ENDPOINT,
        SecretArn: secret.secretArn,
        DistributionId: distribution.distributionId,
        CustomHeaderName: SECRET_CUSTOM_HEADER,
      },
    });

    secret.addRotationSchedule("rotationSchedule", {
      rotationLambda: lambda.Function.fromFunctionArn(
        this,
        "rotationLambda",
        rotation.getAtt("Outputs.RotationLambdaArn").toString()
      ),
      automaticallyAfter: cdk.Duration.days(1),
      rotateImmediatelyOnUpdate: true, // to delegate secret generation to rotation lambda
    });

    const meta = new sam.CfnApplication(this, "MetaURL", {
      location: {
        applicationId:
          "arn:aws:serverlessrepo:us-east-1:015397314665:applications/url-meta",
        semanticVersion: "0.1.3",
      },

      parameters: {
        Name: "Example-url-meta",
        SecureLambdaUrlEndpoint: SECRETS_MANAGER_ENDPOINT,
        SecureLambdaUrlSecretArn: secret.secretArn,
        SecureLambdaUrlHeaderName: SECRET_CUSTOM_HEADER,
        SecureLambdaUrlExtension: extension
          .getAtt("Outputs.LambdaExtensionLayer")
          .toString(),
      },
    });

    distribution.addBehavior(
      "/meta",
      new origins.HttpOrigin(
        meta.getAtt("Outputs.PreviewFunctionUrlDomainName").toString(),
        {
          customHeaders: {
            [SECRET_CUSTOM_HEADER]: cdk.Token.asString(
              secret.secretValue.unsafeUnwrap()
            ),
          },
        }
      ),
      {
        allowedMethods: cloudfront.AllowedMethods.ALLOW_GET_HEAD,
        cachedMethods: cloudfront.CachedMethods.CACHE_GET_HEAD,
        viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
        cachePolicy: new cloudfront.CachePolicy(this, "MetaURLCachePolicy", {
          queryStringBehavior:
            cloudfront.CacheQueryStringBehavior.allowList("urls"),
        }),
      }
    );

    // https://kuwyyqx3wge457l4lqvt5dcxtm0jwihc.lambda-url.eu-west-1.on.aws/

    // Output the CloudFront distribution domain name
    new cdk.CfnOutput(this, "CloudfrontDistribution", {
      value: distribution.distributionDomainName,
    });

    // Output the Lambda function URL
    new cdk.CfnOutput(this, "LambdaFunctionURL", {
      value: lambdaUrl.url,
    });
  }
}
