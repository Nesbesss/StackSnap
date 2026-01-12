package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"net/smtp"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)


var (
	SMTPHost = os.Getenv("SMTP_HOST")
	SMTPPort = os.Getenv("SMTP_PORT")
	SMTPUser = os.Getenv("SMTP_USER")
	SMTPPass = os.Getenv("SMTP_PASS")
	SMTPSend = os.Getenv("SMTP_SEND")
)



type License struct {
	Key       string    `json:"key"`
	Email     string    `json:"email"`
	MachineID string    `json:"machine_id"`
	Plan      string    `json:"plan"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}


var db *sql.DB

func initDB() {
	var err error

	os.MkdirAll("data", 0755)

	db, err = sql.Open("sqlite", "data/licenses.db")
	if err != nil {
		log.Fatal(err)
	}


	query := `
	CREATE TABLE IF NOT EXISTS licenses (
		key TEXT PRIMARY KEY,
		email TEXT,
		machine_id TEXT,
		plan TEXT,
		active INTEGER DEFAULT 1,
		created_at DATETIME
	);
	`
	_, err = db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}



func handleBuy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
		Plan  string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Plan == "" {
		req.Plan = "lifetime"
	}


	key := fmt.Sprintf("STACKSNAP-PRO-%s", strings.ToUpper(uuid.New().String()[0:18]))


	_, err := db.Exec("INSERT INTO licenses (key, email, machine_id, plan, created_at) VALUES (?, ?, ?, ?, ?)",
		key, req.Email, "", req.Plan, time.Now())

	if err != nil {
		log.Printf("DB Error: %v", err)
		http.Error(w, "Failed to generate license", http.StatusInternalServerError)
		return
	}


	sendLicenseEmail(req.Email, key)



	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "License sent to email!",
		"key":     key,
	})
}

func sendLicenseEmail(to, key string) {
	msg := fmt.Sprintf("Subject: Your StackSnap Pro License\r\n"+
		"\r\n"+
		"Hello!\r\n\r\n"+
		"Thank you for choosing StackSnap.\r\n\r\n"+
		"Your License Key: %s\r\n\r\n"+
		"Copy this key into the StackSnap onboarding wizard to get started.\r\n\r\n"+
		"Cheers,\r\nStackSnap Team", key)

	if SMTPHost != "" && SMTPUser != "" {
		auth := smtp.PlainAuth("", SMTPUser, SMTPPass, SMTPHost)
		err := smtp.SendMail(SMTPHost+":"+SMTPPort, auth, SMTPSend, []string{to}, []byte(msg))
		if err != nil {
			log.Printf(" FAILED TO SEND EMAIL: %v", err)
		} else {
			log.Printf(" EMAIL SENT TO: %s", to)
		}
	} else {
		log.Printf(" [MOCK EMAIL] TO: %s | KEY: %s", to, key)
	}
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		LicenseKey string `json:"license_key"`
		MachineID  string `json:"machine_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	log.Printf(" Verifying Key: %s for Machine: %s", req.LicenseKey, req.MachineID)

	var storedMachineID string
	var email string

	err := db.QueryRow("SELECT machine_id, email FROM licenses WHERE key = ?", req.LicenseKey).Scan(&storedMachineID, &email)
	if err == sql.ErrNoRows {
		http.Error(w, `{"valid": false, "reason": "invalid_key"}`, http.StatusOK)
		return
	} else if err != nil {
		log.Printf("DB Error: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}






	if storedMachineID == "" {

		_, err := db.Exec("UPDATE licenses SET machine_id = ? WHERE key = ?", req.MachineID, req.LicenseKey)
		if err != nil {
			log.Printf("Failed to bind license: %v", err)
			http.Error(w, "Activation failed", http.StatusInternalServerError)
			return
		}
		log.Printf(" Activated License %s for %s", req.LicenseKey, email)
		json.NewEncoder(w).Encode(map[string]interface{}{"valid": true, "activated": true})
		return
	}

	if storedMachineID == req.MachineID {
		log.Printf(" Verified License %s", req.LicenseKey)
		json.NewEncoder(w).Encode(map[string]interface{}{"valid": true})
		return
	}


	log.Printf(" License Check Failed: Key used on different machine")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":  false,
		"reason": "already_activated_on_another_machine",
	})
}

func main() {
	initDB()


	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)


	http.HandleFunc("/api/buy", handleBuy)
	http.HandleFunc("/api/verify", handleVerify)

	port := "8081"
	fmt.Printf(" License Server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
