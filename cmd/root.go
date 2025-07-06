package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "turbogo",
		Short: "TurboGo CLI 🚀 - Project generator",
	}

	var newCmd = &cobra.Command{
		Use:   "new [name]",
		Short: "Create a new TurboGo backend project",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RunProjectCreator(args[0])
		},
	}

	rootCmd.AddCommand(newCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func RunProjectCreator(projectName string) {
	base := filepath.Join(".", projectName)

	dirs := []string{
		"cmd",
		"internal/router",
		"internal/controller",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(base, dir), os.ModePerm); err != nil {
			fmt.Println("❌ Gagal membuat folder:", err)
			return
		}
	}

	writeFile(filepath.Join(base, "cmd"), "main.go", GenerateMainFile(projectName, "User"))
	writeFile(filepath.Join(base, "internal/router"), "router.go", GenerateRouter(projectName, "User"))
	writeFile(filepath.Join(base, "internal/controller"), "user.go", GenerateHandlerController("User"))

	if err := runGoModInit(base, projectName); err != nil {
		fmt.Println("❌ Gagal menjalankan go mod init:", err)
		return
	}

	fmt.Println("✅ Project", projectName, "berhasil dibuat!")
}

func writeFile(dir, filename, content string) {
	_ = os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
}

func runGoModInit(baseDir, moduleName string) error {
	cmd := exec.Command("go", "mod", "init", moduleName)
	cmd.Dir = baseDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}