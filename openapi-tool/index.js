#!/usr/bin/env node

const yargs = require('yargs/yargs');
const { hideBin } = require('yargs/helpers');
const { generateFiles } = require('./commands/generate');

const argv = yargs(hideBin(process.argv))
  .command('generate [files..]', 'Generate files', (yargs) => {
    yargs.positional('files', {
      describe: 'List of files to generate',
      type: 'array'
    });
  }, generateFiles)
  .demandCommand(1, 'You need at least one command before moving on')
  .help()
  .argv;
