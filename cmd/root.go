package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func main() {
	var withAuth bool

	var rootCmd = &cobra.Command{
		Use:   "turbogo",
		Short: "🚀 TurboGo CLI - Lightweight backend project generator",
		Long: `TurboGo adalah framework CLI untuk membuat backend ringan dengan cepat.
Fitur tersedia: generator project, auth, dan integrasi modular.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	var newCmd = &cobra.Command{
		Use:   "new [name]",
		Short: "🛠️  Generate a new TurboGo backend project",
		Long:  "Membuat project TurboGo baru dengan struktur dasar dan controller.",
		Example: `
# Buat project biasa
turbogo new myapp

# Buat project dengan endpoint login
turbogo new myapp --with-auth`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RunProjectCreator(args[0], withAuth)
		},
	}

	newCmd.Flags().BoolVar(&withAuth, "with-auth", false, "Tambahkan endpoint login + auth controller")
	rootCmd.AddCommand(newCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func RunProjectCreator(projectName string, withAuth bool) {
	projectName = strings.TrimSpace(projectName)
	base := filepath.Join(".", projectName)

	color.Yellow("📦 Membuat struktur folder untuk '%s'...", projectName)
	dirs := []string{
		"internal/router",
		"internal/controller",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(base, dir)
		if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
			color.Red("❌ Gagal membuat folder: %s", err)
			return
		}
		color.Green("📁 %s", fullPath)
		time.Sleep(150 * time.Millisecond)
	}

	color.Yellow("\n📝 Menulis file-file template...")
	writeFile(filepath.Join(base, "."), "main.go", GenerateMainFile(projectName, "User"))
	writeFile(filepath.Join(base, "internal/router"), "router.go", GenerateRouter(projectName, "User", withAuth))
	writeFile(filepath.Join(base, "internal/controller"), "user.go", GenerateHandlerController("User"))

	if withAuth {
		writeFile(filepath.Join(base, "internal/controller"), "auth.go", GenerateAuthController())
		color.Green("🔐 Fitur auth ditambahkan")
	}

	writeFile(base, ".env", GenerateDotEnv())
	writeFile(base, "README.md", GenerateReadme(projectName))
	writeFile(base, ".gitignore", GenerateGitignore())

	color.Yellow("\n📦 Menjalankan go mod init...")
	if err := runGoModInit(base, projectName); err != nil {
		color.Red("❌ Gagal menjalankan go mod init: %v", err)
		return
	}

	color.Green("✅ Project '%s' berhasil dibuat!", projectName)
	color.Cyan("➡️  Jalankan: cd %s && go run .", projectName)
}

func writeFile(dir, filename, content string) {
	fullPath := filepath.Join(dir, filename)
	_ = os.WriteFile(fullPath, []byte(content), 0644)
	color.Green("✏️  %s", fullPath)
}

func runGoModInit(baseDir, moduleName string) error {
	cmd := exec.Command("go", "mod", "init", moduleName)
	cmd.Dir = baseDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}