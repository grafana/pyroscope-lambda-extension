import yargs from "https://deno.land/x/yargs@v17.5.1-deno/deno.ts";
import * as log from "https://deno.land/std@0.146.0/log/mod.ts";

// Not all regions support multi architecture
// Best way I found to find them is to access
// https://us-east-1.console.aws.amazon.com/lambda/home?region=us-east-1#/create/layer
// per region
const regions = [
  { region: "eu-north-1", arch: false },
  { region: "ap-south-1", arch: true },
  { region: "eu-west-3", arch: false },
  { region: "eu-west-2", arch: true },
  { region: "eu-west-1", arch: true },
  { region: "ap-northeast-3", arch: true },
  { region: "ap-northeast-2", arch: true },
  { region: "ap-northeast-1", arch: true },
  { region: "sa-east-1", arch: false },
  { region: "ca-central-1", arch: false },
  { region: "ap-southeast-1", arch: true },
  { region: "ap-southeast-2", arch: false },
  { region: "eu-central-1", arch: true },
  { region: "us-east-1", arch: true },
  { region: "us-east-2", arch: true },
  { region: "us-west-1", arch: false },
  { region: "us-west-2", arch: true },
];

const y = yargs(Deno.args)
  .options({
    "dry-run": {
      describe: "Run the program without actually invoking commands",
      default: true,
      type: "boolean",
    },
    "name": {
      describe: "The name of the extensoin",
      demandOption: true,
    },
    "table-file": {
      describe: "file with the release info",
      default: "release.tmp.md",
    },
    "log-level": {
      describe: "DEBUG | INFO | WARNING | ERROR | CRITICAL",
      demandOption: true,
      default: "INFO",
      type: "string",
    },
  })
  .parse();

await log.setup({
  handlers: {
    console: new log.handlers.ConsoleHandler(y.logLevel),
  },

  loggers: {
    // configure default logger available via short-hand methods above.
    default: {
      level: y.logLevel,
      handlers: ["console"],
    },

    tasks: {
      level: y.logLevel,
      handlers: ["console"],
    },
  },
});

function getLayerName(name: string, type: "amd" | "arm") {
  switch (type) {
    case "amd": {
      return `${name}-x86_64`;
    }
    case "arm": {
      return `${name}-arm64`;
    }
    default: {
      throw new Error(`Unsupported type: '${type}'`);
    }
  }
}

async function runCommand(cmd: string[], { cwd }: { cwd?: string } = {}) {
  // Since this script is used internally only
  // we don't have to be so strict
  if (y.dryRun === true) {
    return Promise.resolve("");
  } else {
    const p = Deno.run({ cmd, stdout: "piped", cwd: cwd });

    const status = await p.status();
    if (!status.success) {
      Deno.exit(1);
    }
    const output = await p.output();

    return new TextDecoder().decode(output);
  }
}

async function publishCmd(
  layerName: string,
  region: string,
  cwd: string,
): Promise<{ version: string; layernArn: string; fullLayerName: string }> {
  if (y.dryRun) {
    return {
      version: "999",
      layernArn: "arn:aws:lambda:us-east-1:myacc:layer:pyroscope-test-x86_64",
      fullLayerName:
        "arn:aws:lambda:us-east-1:myacc:layer:pyroscope-test-x86_64:999",
    };
  }

  const output = await runCommand([
    "aws",
    "lambda",
    "publish-layer-version",
    `--layer-name=${layerName}`,
    `--region=${region}`,
    `--zip-file=fileb://extension.zip`,
  ], { cwd });

  const parsed = JSON.parse(output);
  return {
    // eg: 1
    version: parsed.Version,
    // eg: 'arn:aws:lambda:us-east-1:myacc:layer:pyroscope-test-x86_64'
    layernArn: parsed.LayerArn,
    // eg: 'arn:aws:lambda:us-east-1:myacc:layer:pyroscope-test-x86_64:999'
    fullLayerName: parsed.LayerVersionArn,
  };
}

async function makePublic(
  { layerName, region, version }: {
    layerName: string;
    region: string;
    version: string;
  },
) {
  if (y.dryRun) {
    return Promise.resolve("");
  }
  const statementId = [layerName, region, version].join("-");
  const output = await runCommand([
    "aws",
    "lambda",
    "add-layer-version-permission",
    `--region=${region}`,
    `--layer-name=${layerName}`,
    `--statement-id=${statementId}`,
    `--version-number=${version}`,
    `--principal=*`,
    `--action=lambda:GetLayerVersion`,
  ]);

  const parsed = JSON.parse(output);
  return {
    // eg: 1
    version: parsed.Version,
    // eg: 'arn:aws:lambda:us-east-1:myacc:layer:pyroscope-test-x86_64'
    layernArn: parsed.layernArn,
    // eg: 'arn:aws:lambda:us-east-1:myacc:layer:pyroscope-test-x86_64:999'
    fullLayerName: parsed.LayerVersionArn,
  };
}

export async function run() {
  log.info("Publishing extension...");
  const amd = await Promise.all(regions.map(async (r) => {
    const cwd = "bin/x86_64";
    const arch = "x86_64";
    const layerName = getLayerName(y.name, "amd");
    const region = r.region;

    log.debug("Publishing", { layerName, region, arch });
    const output = await publishCmd(layerName, region, cwd);
    log.debug("Published", { ...output });

    return { layerName, region, arch, ...output };
  }));
  const arm = await Promise.all(
    regions.filter((r) => r.arch).map(async (r) => {
      const cwd = "bin/arm64";
      const arch = "arm64";
      const layerName = getLayerName(y.name, "arm");
      const region = r.region;

      log.debug("Publishing");
      log.debug({ layerName, region, arch });
      const output = await publishCmd(layerName, region, cwd);
      log.debug("Published");
      log.debug({ ...output });

      return { layerName, region, arch, ...output };
    }),
  );

  const out = [...amd, ...arm];

  log.info("Making extensions public...");
  Promise.all(
    out.map(async ({ layerName, version, region }) => {
      log.debug("Making it public", { layerName, version, region });
      const output = await makePublic({ layerName, version, region });
      log.debug("Done.");
      log.debug({ layerName, version, region });

      return;
    }),
  );

  return out.map(({ region, arch, fullLayerName }) => {
    return {
      region,
      arch,
      fullLayerName,
    };
  });
}

export function generateReleaseTable(published: {
  region: string;
  arch: string;
  fullLayerName: string;
}[]): string {
  return `
| region                   | arch | layer name |
|--------------------------|------|------------|\n` +
    published.map((a) => {
      return `|\`${a.region}\`|\`${a.arch}\`|\`${a.fullLayerName}\`|`;
    }).sort().join("\n");
}

// am I being executed or imported?
if (import.meta.main) {
  const output = await run();

  log.info("Generating CHANGELOG...");
  const releaseTable = generateReleaseTable(output);
  if (y.dryRun) {
    console.log(releaseTable);
  } else {
    await Deno.writeTextFile(y.tableFile, releaseTable);
  }
}
