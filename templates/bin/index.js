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
  .description(
    "Scaffold a Golang project using the TurboGo Framework with Feature-Based Architecture"
  )
  .argument("[project-name]", "Project name")
  .action(async (projectName) => {
    printBanner();
    console.log(
      chalk.blue.bold("🚀 Welcome to the TurboGo Project Generator!")
    );
    console.log(
      chalk.gray("   Using Feature-Based Architecture for better scalability\n")
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

    const { featureName } = await inquirer.prompt([
      {
        type: "input",
        name: "featureName",
        message: "Initial feature name (PascalCase):",
        default: "User",
        validate: (val) =>
          /^[A-Z][a-zA-Z0-9]*$/.test(val) ||
          "Use PascalCase format (e.g., User, Product, Auth)",
      },
    ]);

    const { wantMoreFeatures } = await inquirer.prompt([
      {
        type: "confirm",
        name: "wantMoreFeatures",
        message: "Do you want to add more features?",
        default: false,
      },
    ]);

    let additionalFeatures = [];
    if (wantMoreFeatures) {
      const { featuresInput } = await inquirer.prompt([
        {
          type: "input",
          name: "featuresInput",
          message:
            "Enter additional feature names (comma-separated, PascalCase):",
          default: "",
          validate: (input) => {
            if (!input.trim()) return true;
            const features = input
              .split(",")
              .map((f) => f.trim())
              .filter((f) => f.length > 0);
            return (
              features.every((f) => /^[A-Z][a-zA-Z0-9]*$/.test(f)) ||
              "All features must be in PascalCase format (e.g., Auth, Product, Order)"
            );
          },
        },
      ]);

      if (featuresInput.trim()) {
        additionalFeatures = featuresInput
          .split(",")
          .map((f) => f.trim())
          .filter((f) => f.length > 0);
      }
    }

    const { useDatabase } = await inquirer.prompt([
      {
        type: "confirm",
        name: "useDatabase",
        message: "Will you use a database?",
        default: true,
      },
    ]);

    let dbConfig = null;
    if (useDatabase) {
      dbConfig = await inquirer.prompt([
        {
          type: "list",
          name: "type",
          message: "Select database type:",
          choices: ["PostgreSQL", "MySQL", "SQLite", "MongoDB"],
          default: "PostgreSQL",
        },
      ]);
    }

    const config = {
      name: featureName,
      features: [featureName, ...additionalFeatures],
      database: useDatabase ? dbConfig : null,
    };

    const spinner = ora(
      "📦 Generating project with feature-based architecture..."
    ).start();

    try {
      await generateProject(projectName, projectPath, config);
      spinner.succeed(chalk.green("✅ Project created successfully!\n"));

      console.log(chalk.blue.bold("📁 Project Structure:"));
      console.log(chalk.gray(`${projectName}/`));
      console.log(chalk.gray(`├── features/`));
      config.features.forEach((feature, index) => {
        const isLast = index === config.features.length - 1;
        const prefix = isLast ? "└──" : "├──";
        console.log(chalk.gray(`│   ${prefix} ${feature.toLowerCase()}/`));
        console.log(
          chalk.gray(`│   ${isLast ? "    " : "│   "}├── handler.go`)
        );
        console.log(
          chalk.gray(`│   ${isLast ? "    " : "│   "}├── service.go`)
        );
        console.log(
          chalk.gray(`│   ${isLast ? "    " : "│   "}├── repository.go`)
        );
        console.log(chalk.gray(`│   ${isLast ? "    " : "│   "}└── router.go`));
      });
      console.log(chalk.gray(`├── main.go`));
      console.log(chalk.gray(`├── .env`));
      console.log(chalk.gray(`├── .gitignore`));
      console.log(chalk.gray(`└── README.md\n`));

      console.log(chalk.blue.bold("🎯 Next Steps:"));
      console.log(chalk.cyan(`  1. cd ${projectName}`));
      console.log(chalk.cyan(`  2. go mod tidy`));
      console.log(chalk.cyan(`  3. Configure your .env file`));
      if (useDatabase) {
        console.log(chalk.cyan(`  4. Setup your ${dbConfig.type} database`));
        console.log(chalk.cyan(`  5. go run .`));
      } else {
        console.log(chalk.cyan(`  4. go run .`));
      }

      console.log(chalk.yellow("\n💡 Tips:"));
      console.log(
        chalk.gray("  • Each feature is self-contained in its own folder")
      );
      console.log(
        chalk.gray("  • Add new features by creating folders in features/")
      );
      console.log(
        chalk.gray("  • Follow the handler → service → repository pattern")
      );

      console.log(
        chalk.green(
          "\n✨ Happy coding with TurboGo Feature-Based Architecture! 🚀"
        )
      );

      if (config.features.length > 1) {
        console.log(
          chalk.blue(
            `\n📦 Generated ${config.features.length} features: ${config.features.join(", ")}`
          )
        );
      }
    } catch (error) {
      spinner.fail(chalk.red("❌ Failed to generate project"));
      console.error(error);
      process.exit(1);
    }
  });

program
  .command("add-feature")
  .description("Add a new feature to existing TurboGo project")
  .action(async () => {
    console.log(chalk.blue.bold("\n➕ Add New Feature to TurboGo Project\n"));

    const { featureName } = await inquirer.prompt([
      {
        type: "input",
        name: "featureName",
        message: "Feature name (PascalCase):",
        validate: (val) =>
          /^[A-Z][a-zA-Z0-9]*$/.test(val) ||
          "Use PascalCase format (e.g., Product, Order, Payment)",
      },
    ]);

    const featuresDir = path.join(process.cwd(), "features");

    if (!(await fs.pathExists(featuresDir))) {
      console.log(chalk.red("❌ This doesn't appear to be a TurboGo project."));
      console.log(
        chalk.gray("   Run this command from the root of your TurboGo project.")
      );
      process.exit(1);
    }

    const featureDir = path.join(featuresDir, featureName.toLowerCase());

    if (await fs.pathExists(featureDir)) {
      console.log(chalk.red(`❌ Feature "${featureName}" already exists!`));
      process.exit(1);
    }

    const mainGoPath = path.join(process.cwd(), "main.go");
    if (!(await fs.pathExists(mainGoPath))) {
      console.log(chalk.red("❌ Cannot find main.go in current directory."));
      process.exit(1);
    }

    const mainGoContent = await fs.readFile(mainGoPath, "utf-8");
    const projectNameMatch = mainGoContent.match(
      /import[^)]*"([^"]+)\/features/
    );

    if (!projectNameMatch) {
      console.log(chalk.red("❌ Cannot detect project name from main.go"));
      process.exit(1);
    }

    const projectName = projectNameMatch[1];

    const spinner = ora(`Creating feature "${featureName}"...`).start();

    try {
      const { addFeature, updateReadmeWithFeature } = await import(
        "../lib/generate.js"
      );

      await addFeature(featureName, projectName, featureDir);

      const featureLower = featureName.toLowerCase();
      const importLine = `\t"${projectName}/features/${featureLower}"`;
      const registerLine = `\t${featureLower}.RegisterRoutes(app)`;

      let updatedMainGo = mainGoContent;

      if (!mainGoContent.includes(importLine)) {
        const importBlockMatch = mainGoContent.match(
          /import \(\n([\s\S]*?)\n\)/
        );
        if (importBlockMatch) {
          const imports = importBlockMatch[1];
          const lastFeatureImport = imports
            .split("\n")
            .filter((line) => line.includes("/features/"))
            .pop();

          if (lastFeatureImport) {
            updatedMainGo = updatedMainGo.replace(
              lastFeatureImport,
              `${lastFeatureImport}\n${importLine}`
            );
          } else {
            updatedMainGo = updatedMainGo.replace(
              /("github\.com\/Dziqha\/TurboGo")/,
              `$1\n${importLine}`
            );
          }
        }
      }

      if (!mainGoContent.includes(registerLine)) {
        const lastRegister = updatedMainGo
          .split("\n")
          .filter((line) => line.includes(".RegisterRoutes(app)"))
          .pop();

        if (lastRegister) {
          updatedMainGo = updatedMainGo.replace(
            lastRegister,
            `${lastRegister}\n${registerLine}`
          );
        } else {
          updatedMainGo = updatedMainGo.replace(
            /app := TurboGo\.New\(\)/,
            `app := TurboGo.New()\n\t\n\t// Register feature routes\n${registerLine}`
          );
        }
      }

      await fs.writeFile(mainGoPath, updatedMainGo);

      const readmePath = path.join(process.cwd(), "README.md");
      if (await fs.pathExists(readmePath)) {
        await updateReadmeWithFeature(readmePath, featureName);
      }

      spinner.succeed(
        chalk.green(`✅ Feature "${featureName}" created successfully!\n`)
      );

      console.log(chalk.blue.bold("📁 Generated Files:"));
      console.log(chalk.gray(`features/${featureLower}/`));
      console.log(chalk.gray(`├── handler.go`));
      console.log(chalk.gray(`├── service.go`));
      console.log(chalk.gray(`├── repository.go`));
      console.log(chalk.gray(`└── router.go`));

      console.log(chalk.green("\n✅ Updated:"));
      console.log(chalk.gray(`main.go (added import and route registration)`));
      console.log(
        chalk.gray(`README.md (added project structure and API endpoints)`)
      );

      console.log(chalk.blue.bold("\n🔗 Available Endpoints:"));
      console.log(chalk.cyan(`GET    /api/${featureLower}`));
      console.log(chalk.cyan(`POST   /api/${featureLower}`));
      console.log(chalk.cyan(`GET    /api/${featureLower}/:id`));
      console.log(chalk.cyan(`PUT    /api/${featureLower}/:id`));
      console.log(chalk.cyan(`DELETE /api/${featureLower}/:id`));

      console.log(chalk.yellow("\n💡 Next Steps:"));
      console.log(
        chalk.gray(
          `  1. Implement business logic in features/${featureLower}/service.go`
        )
      );
      console.log(
        chalk.gray(
          `  2. Add database queries in features/${featureLower}/repository.go`
        )
      );
      console.log(
        chalk.gray(
          `  3. Customize handlers in features/${featureLower}/handler.go`
        )
      );
      console.log(chalk.gray(`  4. Run: go run .`));
    } catch (error) {
      spinner.fail(chalk.red("❌ Failed to create feature"));
      console.error(error);
      process.exit(1);
    }
  });

program.parse();
