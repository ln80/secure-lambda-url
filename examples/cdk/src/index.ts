import { APIGatewayProxyEventV2, APIGatewayProxyHandlerV2 } from "aws-lambda";

const SECURE_LAMBDA_URL_PORT =
  process.env.SECURE_LAMBDA_URL_HTTP_PORT || "3579";

const SECURE_HEADER_NAME = process.env.SECURE_LAMBDA_URL_HEADER_NAME;
if (!SECURE_HEADER_NAME) {
  throw new Error("secure header name env var missed");
}

export const handler: APIGatewayProxyHandlerV2 = async (
  event: APIGatewayProxyEventV2
) => {
  let status = 200;
  try {
    const headerValue = event.headers?.[SECURE_HEADER_NAME.toLowerCase()] || "";
    if (!headerValue) {
      throw new Error("secure header value missed");
    }
    const response = await fetch(
      `http://localhost:${SECURE_LAMBDA_URL_PORT}?key=${encodeURIComponent(
        headerValue
      )}`,
      {
        method: "GET",
        headers: {
          "X-Aws-Token": process.env.AWS_SESSION_TOKEN!,
        },
      }
    );
    status = response.status;
  } catch (err: any) {
    console.error("Secure Lambda URL IPC call failed", err);
    status = 500;
  }

  if (status >= 400) {
    const error: { [key: number]: string } = {
      500: "Internal error",
      401: "Unauthorized",
      400: "Bad request",
    };

    return {
      statusCode: status,
      body: JSON.stringify({ message: error[status] || error[400] }),
    };
  }

  return {
    statusCode: 200,
    body: JSON.stringify({ message: "Example Secure Lambda URL" }),
  };
};
