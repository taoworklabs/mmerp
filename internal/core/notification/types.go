package notification

// Kind is what happened; readers build the text from it, so a notification copies nothing of its record.
type Kind string

const (
	ApprovalRequested Kind = "approval_requested"
	ApprovalApproved  Kind = "approval_approved"
	ApprovalRejected  Kind = "approval_rejected"
	Mentioned         Kind = "mentioned"
	JobCompleted      Kind = "job_completed"
	JobFailed         Kind = "job_failed"
)

// emailed says which kinds are emailed too: a job's requester is waiting in the app.
func (k Kind) emailed() bool { return k != JobCompleted && k != JobFailed }

const pageSize = 50

// Notification is one of the caller's notifications. RecordID, Label and ActorName are
// only there while the caller may view the record.
type Notification struct {
	ID         int64   `json:"id"`
	Kind       string  `json:"kind" enum:"approval_requested,approval_approved,approval_rejected,mentioned,job_completed,job_failed"`
	RecordType *string `json:"record_type" doc:"e.g. hrm.leave_request; null for a job"`
	RecordID   *int64  `json:"record_id"`
	Label      *string `json:"label" doc:"The document number; null for a catalogue"`
	ActorName  *string `json:"actor_name"`
	JobID      *int64  `json:"job_id"`
	CreatedAt  string  `json:"created_at" format:"date-time"`
	Read       bool    `json:"read"`
}

type Page struct {
	Items []Notification `json:"items" nullable:"false"`
	// Pass as before to read the next, older page; null on the last one.
	NextBefore *int64 `json:"next_before"`
}

// MailServer is the tenant's SMTP server as read back: never its password.
type MailServer struct {
	Host        string `json:"host"`
	Port        int32  `json:"port"`
	Security    string `json:"security" enum:"starttls,tls,none"`
	Username    string `json:"username"`
	PasswordSet bool   `json:"password_set"`
	FromAddress string `json:"from_address"`
	BaseURL     string `json:"base_url" doc:"Where users reach mmerp; email links start with it"`
}

type MailServerInput struct {
	Host     string  `json:"host" minLength:"1" maxLength:"255"`
	Port     int32   `json:"port" minimum:"1" maximum:"65535"`
	Security string  `json:"security" enum:"starttls,tls,none"`
	Username string  `json:"username" maxLength:"255"`
	Password *string `json:"password" maxLength:"255" doc:"Null keeps the stored one"`
	// A bare address, e.g. mmerp@example.com.
	FromAddress string `json:"from_address" maxLength:"254"`
	BaseURL     string `json:"base_url" maxLength:"255"`
}
