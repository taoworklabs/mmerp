package hrm_test

import (
	"fmt"
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// 2026-03-09 to 13 is a Monday-to-Friday week.
func (f *leaveFixture) leaveOf(typeID int64, start, end, days string) hrm.LeaveFields {
	l := f.fields(start, end, days)
	l.LeaveTypeID = typeID
	return l
}

func TestLeaveTypeCapsOneRequest(t *testing.T) {
	f := newLeaveFixture(t)
	wedding := must(t)(f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "Kết hôn", Paid: true, Active: true, MaxDays: str("3")}))

	_, err := f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.leaveOf(wedding, "2026-03-09", "2026-03-13", "4")})
	wantErr(t, err, "leave_days_exceed_type_max")
	if e, _ := err.(*platform.Error); e.Params["max_days"] != "3" {
		t.Fatalf("params = %v, want max_days 3", e.Params)
	}

	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.leaveOf(wedding, "2026-03-09", "2026-03-11", "3")}))
	wantErr(t, f.hrm.UpdateLeave(f.e, id, hrm.LeaveUpdate{Version: 1, LeaveFields: f.leaveOf(wedding, "2026-03-09", "2026-03-12", "3.5")}), "leave_days_exceed_type_max")

	// A type without a cap is bounded only by the working days.
	must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-16", "2026-03-20", "5")}))
}

func TestTighteningACapRefusesTheDraftOnApproval(t *testing.T) {
	f := newLeaveFixture(t)
	in := hrm.LeaveTypeInput{Name: "Kết hôn", Paid: true, Active: true, MaxDays: str("3")}
	wedding := must(t)(f.hrm.SaveLeaveType(f.hr, 0, in))
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.leaveOf(wedding, "2026-03-09", "2026-03-11", "3")}))

	in.MaxDays = str("2")
	must(t)(f.hrm.SaveLeaveType(f.hr, wedding, in))
	wantErr(t, f.send(f.e, id), "leave_days_exceed_type_max")
	if s := f.leave(id).Status; s != "draft" {
		t.Fatalf("status = %s, want draft", s)
	}
}

// attach records a file on the request as the attachment module would; the contents
// do not matter here, only that a row names the request.
func (f *leaveFixture) attach(id int64) {
	f.t.Helper()
	_, err := f.system.Exec(f.admin, `INSERT INTO attachment.files (doc_type, doc_id, file_id, name, size, content_type, uploaded_by)
		SELECT 'hrm.leave_request', $1, $2, 'giay-kham.pdf', 1, 'application/pdf', id FROM iam.users WHERE login = 'e'`, id, fmt.Sprint("f", id))
	f.check(err)
}

func TestSendingNeedsTheAttachmentTheKindRequires(t *testing.T) {
	f := newLeaveFixture(t)
	sick := must(t)(f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "Ốm", Active: true, RequiresAttachment: true}))
	// A draft saves without a file.
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.leaveOf(sick, "2026-03-09", "2026-03-10", "2")}))
	wantErr(t, f.send(f.e, id), "leave_attachment_required")
	if s := f.leave(id).Status; s != "draft" {
		t.Fatalf("status = %s, want draft", s)
	}
	f.attach(id)
	f.check(f.send(f.e, id))
	if s := f.leave(id).Status; s != "posted" {
		t.Fatalf("status = %s, want posted", s)
	}
}

func TestApprovalNeedsTheAttachmentStillThere(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	sick := must(t)(f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "Ốm", Active: true, RequiresAttachment: true}))
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.leaveOf(sick, "2026-03-09", "2026-03-10", "2")}))
	f.attach(id)
	f.check(f.send(f.e, id))
	if _, err := f.system.Exec(f.admin, `DELETE FROM attachment.files WHERE doc_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	wantErr(t, f.appr.Approve(f.m, f.ticket(id), 1), "leave_attachment_required")
	if s := f.leave(id).Status; s != "pending_approval" {
		t.Fatalf("status = %s, want pending_approval", s)
	}
}

// Kind names are unique regardless of case, on create and on rename.
func TestKindNameTaken(t *testing.T) {
	f := newLeaveFixture(t)
	must(t)(f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "Kết hôn", Active: true}))
	other := must(t)(f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "Ốm", Active: true}))
	_, err := f.hrm.SaveLeaveType(f.hr, 0, hrm.LeaveTypeInput{Name: "KẾT HÔN", Active: true})
	wantErr(t, err, "leave_type_taken")
	_, err = f.hrm.SaveLeaveType(f.hr, other, hrm.LeaveTypeInput{Name: "kết hôn", Active: true})
	wantErr(t, err, "leave_type_taken")

	must(t)(f.hrm.SaveContractType(f.hr, 0, hrm.ContractTypeInput{Name: "Thử việc", Active: true}))
	_, err = f.hrm.SaveContractType(f.hr, 0, hrm.ContractTypeInput{Name: "THỬ VIỆC", Active: true})
	wantErr(t, err, "contract_type_taken")
}
