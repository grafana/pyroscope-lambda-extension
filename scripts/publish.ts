import yargs from "https://deno.land/x/yargs@v17.5.1-deno/deno.ts";
import * as log from "https://deno.land/std@0.146.0/log/mod.ts";

type Arch = "x86_64" | "arm64";
// Not all regions support multi architecture
// Best way I found to find them is to access
// https://us-east-1.console.aws.amazon.com/lambda/home?region=us-east-1#/create/layer
// per region
const regions: { region: string; archs: Arch[] }[] = [
  { region: "eu-north-1", archs: [] },
  { region: "ap-south-1", archs: ["x86_64", "arm64"] },
  { region: "eu-west-3", archs: [] },
  { region: "eu-west-2", archs: ["x86_64", "arm64"] },
  { region: "eu-west-1", archs: ["x86_64", "arm64"] },
  { region: "ap-northeast-3", archs: [] },
  { region: "ap-northeast-2", archs: [] },
  { region: "ap-northeast-1", archs: ["x86_64", "arm64"] },
  { region: "sa-east-1", archs: [] },
  { region: "ca-central-1", archs: [] },
  { region: "ap-southeast-1", archs: ["x86_64", "arm64"] },
  { region: "ap-southeast-2", archs: ["x86_64", "arm64"] },
  { region: "eu-central-1", archs: ["x86_64", "arm64"] },
  { region: "us-east-1", archs: ["x86_64", "arm64"] },
  { region: "us-east-2", archs: ["x86_64", "arm64"] },
  { region: "us-west-1", archs: [] },
  { region: "us-west-2", archs: ["x86_64", "arm64"] },
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

function getLayerName(name: string, arch: Arch) {
  switch (arch) {
    case "x86_64": {
      return `${name}-x86_64`;
    }
    case "arm64": {
      return `${name}-arm64`;
    }
    default: {
      throw new Error(`Unsupported arch: '${arch}'`);
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
  arch?: string,
): Promise<{ version: string; layernArn: string; fullLayerName: string }> {
  if (y.dryRun) {
    return {
      version: "999",
      layernArn: "arn:aws:lambda:us-east-1:myacc:layer:pyroscope-test-x86_64",
      fullLayerName:
        "arn:aws:lambda:us-east-1:myacc:layer:pyroscope-test-x86_64:999",
    };
  }

  let cmd = [
    "aws",
    "lambda",
    "publish-layer-version",
    `--layer-name=${layerName}`,
    `--region=${region}`,
    `--zip-file=fileb://extension.zip`,
  ];

  if (arch) {
    cmd = cmd.concat([`--compatible-architectures=${arch}`]);
  }

  const output = await runCommand(cmd, { cwd });

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

async function publishAmd(
  name: string,
  region: typeof regions[number]["region"],
  withArch?: boolean,
): Promise<{
  layerName: string;
  version: string;
  layernArn: string;
  fullLayerName: string;
}> {
  const cwd = "bin/x86_64";
  const layerName = getLayerName(name, "x86_64");
  if (y.dryRun) {
    return Promise.resolve({
      layerName,
      version: "999",
      layernArn: `arn:aws:lambda:${region}:myacc:layer:${layerName}`,
      fullLayerName: `arn:aws:lambda:${region}:myacc:layer:${layerName}:999`,
    });
  }

  // For zones that only support x86_64, we don't pass an architecture
  return {
    ...(await publishCmd(
      layerName,
      region,
      cwd,
      withArch ? "x86_64" : undefined,
    )),
    layerName,
  };
}

async function publishArm(
  name: string,
  region: typeof regions[number]["region"],
): Promise<{
  layerName: string;
  version: string;
  layernArn: string;
  fullLayerName: string;
}> {
  const cwd = "bin/arm64";
  const layerName = getLayerName(name, "arm64");
  if (y.dryRun) {
    return Promise.resolve({
      layerName,
      version: "999",
      layernArn: `arn:aws:lambda:${region}:myacc:layer:${layerName}`,
      fullLayerName: `arn:aws:lambda:${region}:myacc:layer:${layerName}:999`,
    });
  }

  return { ...(await publishCmd(layerName, region, cwd, "arm64")), layerName };
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

  const all = (await Promise.all(regions.map(async (r) => {
    log.debug(r);

    // Since there's only a single architecture in this region
    // We don't ask to specificy a specific arch
    if (!r.archs.length) {
      log.debug("Region has no arch, defaulting to x86_64");
      const amd = await publishAmd(y.name, r.region, false);
      return [{ ...amd, region: r.region, arch: "x86_64" }];
    }

    return await Promise.all(r.archs.map(async (arch) => {
      switch (arch) {
        case "x86_64": {
          log.debug("Publishing x86_64");
          return {
            ...(await publishAmd(y.name, r.region, true)),
            region: r.region,
            arch,
          };
        }
        case "arm64": {
          log.debug("Publishing arm64");
          return {
            ...(await publishArm(y.name, r.region)),
            region: r.region,
            arch,
          };
        }
        default: {
          throw new Error(`Invalid arch ${arch}`);
        }
      }
    }));
  }))).flat();

  log.info("Making extensions public...");
  Promise.all(
    all.map(async ({ layerName, version, region }) => {
      log.debug("Making it public");
      log.debug({ layerName, version, region });
      const output = await makePublic({ layerName, version, region });
      log.debug("Done.");
      log.debug({ layerName, version, region });

      return;
    }),
  );

  return all.map(({ region, arch, fullLayerName }) => {
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
    log.debug(releaseTable);
    await Deno.writeTextFile(y.tableFile, releaseTable);
  }
}
