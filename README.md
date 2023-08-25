# Secure-Lambda-URL

A collection of serverless components to easily protect public lambda function URLs. It includes:

- **Secret Rotation Lambda**, which optionally updates Cloudfront distribution's origin custom header.
- **Lambda Extension**, which simplifies and reduces the boilerplate related to authorization logic.


Both components are distributed as AWS Serverless application models (SAM) and hosted in the serverless application repository (SAR).


### TODO (TDB):
- Collect Cloudwatch authorization-related metrics (customs) at the Lambda extension level.
- Improve testing coverage.
