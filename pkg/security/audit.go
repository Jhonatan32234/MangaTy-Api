package security

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type AuditLog struct {
	Timestamp string `json:"timestamp"`
	UserID    string `json:"user_id,omitempty"`
	IP        string `json:"ip"`
	Action    string `json:"action"`
	Endpoint  string `json:"endpoint"`
	Method    string `json:"method"`
	Status    int    `json:"status"`
	Details   string `json:"details,omitempty"`
}

var (
	auditLogger *log.Logger
	auditFile   *os.File
)

func InitAuditLog() error {
	var err error
	auditFile, err = os.OpenFile("audit.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	auditLogger = log.New(auditFile, "", 0)
	return nil
}

func LogAudit(log AuditLog) {
	log.Timestamp = time.Now().UTC().Format(time.RFC3339)
	
	entry, err := json.Marshal(log)
	if err != nil {
		return
	}
	
	if auditLogger != nil {
		auditLogger.Println(string(entry))
	}
	
	// También loggear a consola en desarrollo
	fmt.Printf("[AUDIT] %s | %s | %s | %s %s | %d\n",
		log.Timestamp, log.UserID, log.IP, log.Method, log.Endpoint, log.Status)
}

func CloseAuditLog() {
	if auditFile != nil {
		auditFile.Close()
	}
}