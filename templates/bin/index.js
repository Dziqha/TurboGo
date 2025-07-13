#!/usr/bin/env node
import inquirer from "inquirer";
import { Command } from "commander";
import chalk from "chalk";
import ora from "ora";
import fs from "fs-extra";
import path from "path";
import { generateProject } from "../lib/generate.js";
import { printBanner } from "../utils/banner.js";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { version } = require("../package.json");

const program = new Command();

program
  .name("create-turbogo")
  .usage("[project-name]")
  .version(version, "-v, --version", "Show CLI version")
  .helpOption("-h, --help", "Display help information")
  .description("Scaffold a Golang project using the TurboGo Framework")
  .argument("[project-name]", "Project name")
  .action(async (projectName) => {
    printBanner();
    console.log(
      chalk.blue.bold("🚀 Welcome to the TurboGo Project Generator!")
    );

    if (!projectName) {
      const { projectName: name } = await inquirer.prompt([
        {
          type: "input",
          name: "projectName",
          message: "Project name:",
          default: "my-turbogo-app",
          validate: (input) =>
            /^[a-zA-Z0-9-_]+$/.test(input) || "Project name is invalid",
        },
      ]);
      projectName = name;
    }

    const projectPath = path.join(process.cwd(), projectName);
    if (await fs.pathExists(projectPath)) {
      console.log(chalk.red(`❌ Folder "${projectName}" already exists!`));
      process.exit(1);
    }

    const { controllerName } = await inquirer.prompt([
      {
        type: "input",
        name: "controllerName",
        message: "Main controller name (CamelCase):",
        default: "Hello",
        validate: (val) =>
          /^[A-Z][a-zA-Z0-9]*$/.test(val) ||
          "Use CamelCase format (e.g., Hello, UserPost)",
      },
    ]);

    const config = {
      name: controllerName,
    };

    const spinner = ora("📦 Generating project...").start();

    try {
      await generateProject(projectName, projectPath, config);
      spinner.succeed(chalk.green("✅ Project created successfully!\n"));

      console.log(chalk.gray("Next steps:"));
      console.log(chalk.cyan(`  cd ${projectName}`));
      console.log(chalk.cyan(`  go run .`));
      console.log(
        chalk.yellow(
          `\n⚠️  Don't forget to adjust your .env file and install any additional Go dependencies.`
        )
      );
      console.log(chalk.gray("\nHappy coding with TurboGo! 🚀"));
    } catch (error) {
      spinner.fail(chalk.red("❌ Failed to generate project"));
      console.error(error);
      process.exit(1);
    }
  });

program.parse();
