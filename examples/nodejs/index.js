const Pyroscope = require("@pyroscope/nodejs");

Pyroscope.init({
  serverAddress: "http://localhost:4040",
  appName: "my-node-service",
});
Pyroscope.start();

function doWork(number) {
  for (let i = 0; i < number; i++) {}
}

exports.handler = async (event, context) => {
  try {
    response = {
      "statusCode": 200,
      "body": JSON.stringify({
        message: "hello world",
      }),
    };
  } catch (err) {
    console.log(err);
    return err;
  }

  doWork(99999999);
  doWork(99999999);

  return response;
};
