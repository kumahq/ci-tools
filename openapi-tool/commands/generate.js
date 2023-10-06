const yaml = require("js-yaml");
const fg = require('fast-glob');
const {createInterface} = require('readline')

const {
    formatProblems,
    getTotals,
    coreVersion,
    loadConfig,
    bundle,
} = require('@redocly/openapi-core');
const {merger} = require('oas-toolkit');
const fs = require("fs"); // Replace with actual function from OAS Toolkit

exports.generateFiles = async (argv) => {
    const files = argv.files;

    if (!files || files.length === 0) {
        console.error('No files provided.');
        return;
    }

    const config = await loadConfig({
        // configPath: redocConfigurationPath
    });
    let hasProblems = false;
    let expandedFiles = await fg.glob(files)
    let apis = (await Promise.all(expandedFiles.map(async (filename) => {
        // Check it's a openapi spec
        if (!await fileIsOpenApiSpec(filename)) {
            return null;
        }
        // Bundle each file independently
        const {
            bundle: result,
            problems,
            ...meta
        } = await bundle({
            config: config,
            ref: filename,
            dereference: false,
            removeUnusedComponents: false,
        });
        if (problems.length > 0) {
            formatProblems(problems, {totals: getTotals(problems), version: coreVersion})
            hasProblems = true;
            // Return proactively if there are problems
            return null;
        }
        const spec = result.parsed

        // Resolve `schema.json` bits
        const name = spec.info['x-ref-schema-name']
        if (name) {
            if (spec.components.schemas[`${name}Item`]?.["$ref"] === "#/components/schemas/schema") {
                spec.components.schemas[`${name}Item`] = spec.components.schemas.schema
                delete spec.components.schemas.schema
            }
            deepReplace(spec, "$ref", "#/components/schemas/schema", `#/components/schemas/${name}Item`)
        }
        return spec
    }))).filter((n) => n !== null);
    if (hasProblems) {
        throw Error("Problems when bundling, not trying to merge")
    }
    console.log(yaml.dump(merger(apis)));
};

function deepReplace(obj, onKey, from, to) {
    for (let key in obj) {
        if (key === onKey && obj[key] === from) {
            obj[key] = to
        } else if (typeof obj[key] === 'object' && obj[key] !== null) {
            // If the current property is an object, recurse into it
            deepReplace(obj[key], onKey, from, to);
        }
    }
}

async function fileIsOpenApiSpec(path) {
    const inputStream = fs.createReadStream(path);
    try {
        for await (const line of createInterface(inputStream)) {
            if (line.match(/^openapi:\s*[0-9]+\.[0-9]+\.[0-9]+/)) {
                return true;
            }
        }
        return false;
    } finally {
        inputStream.destroy(); // Destroy file stream.
    }
}
