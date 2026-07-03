// Command admin is the operator escape hatch for account recovery.
//
// A self-hosted deployment has no email server, so when every admin is locked
// out there is no self-service path back in. Whoever can run this command on
// the server already owns the database, so no extra auth is required.
//
// Usage (inside the backend container):
//
//	smartscan-admin reset-password <email>
//	smartscan-admin list-users
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/database"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
)

func main() {
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cfg := config.Load()
	db, err := database.Connect(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "reset-password":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "Usage: smartscan-admin reset-password <email>")
			os.Exit(1)
		}
		resetPassword(db, os.Args[2])
	case "list-users":
		listUsers(db)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `smartscan-admin — operator account recovery

Commands:
  reset-password <email>   Set a one-time password for the user; they must change it on next login
  list-users               List all staff accounts`)
}

func resetPassword(db *gorm.DB, email string) {
	email = utils.NormalizeEmail(strings.TrimSpace(email))

	var user models.User
	if err := db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error; err != nil {
		fmt.Fprintf(os.Stderr, "No user found with email %q\n", email)
		os.Exit(1)
	}

	tempPassword, err := utils.GenerateSecureTempPassword(12)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate password: %v\n", err)
		os.Exit(1)
	}
	hashed, err := utils.HashPassword(tempPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to hash password: %v\n", err)
		os.Exit(1)
	}

	if err := db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"password_hash":        hashed,
		"must_change_password": true,
		"status":               "active",
	}).Error; err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update password: %v\n", err)
		os.Exit(1)
	}

	// Best-effort token revocation (needs Redis; fails open when unavailable)
	_ = utils.NewTokenBlacklist().RevokeUserTokens(user.ID.String(), 168*time.Hour)

	fmt.Printf("Password reset for %s\n", email)
	fmt.Printf("One-time password: %s\n", tempPassword)
	fmt.Println("The user must change it on next login.")
}

func listUsers(db *gorm.DB) {
	var staff []models.TenantStaff
	if err := db.Preload("User").Order("created_at ASC").Find(&staff).Error; err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list users: %v\n", err)
		os.Exit(1)
	}
	if len(staff) == 0 {
		fmt.Println("No staff accounts found.")
		return
	}
	fmt.Printf("%-40s %-18s %-8s %s\n", "EMAIL", "ROLE", "STATUS", "NAME")
	for _, s := range staff {
		email, status := "-", "-"
		if s.User != nil {
			email = s.User.Email
			status = string(s.User.Status)
		}
		fmt.Printf("%-40s %-18s %-8s %s\n", email, s.Role, status, s.FullName)
	}
}
