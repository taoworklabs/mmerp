package hrm

// Layouts of the official forms, verbatim. A new version of a form is a new value, never an edit.

// layoutD02LT_QD366_2026 is D02-LT of Quyết định 366/QĐ-BHXH, 29/4/2026.
var layoutD02LT_QD366_2026 = []string{
	"STT", "Họ và tên", "(1) Mã số BHXH", "(2) Ngày/ tháng/ năm sinh",
	"(3) Chức vụ, chức danh nghề, nghề nghiệp, nơi làm việc", "(4) Hệ số lương/ Tiền lương",
	"(5) Hệ số chênh lệch bảo lưu", "(6) Phụ cấp chức vụ", "(7) Phụ cấp thâm niên VK (%)",
	"(8) Phụ cấp thâm niên nghề (%)", "(9) Các khoản phụ cấp lương và thu nhập bổ sung",
	"(10) HĐLĐ không xác định thời hạn", "(11) HĐLĐ xác định thời hạn", "(12) HĐ, thỏa thuận khác",
	"(13) Ngày Quyết định, HĐLĐ, HĐLV có hiệu lực", "(14) Từ tháng/ năm", "(15) Đến tháng/ năm",
	"(16) Nơi đăng ký KCB ban đầu", "(17) Hình thức nhận kết quả", "(18) Loại tăng, giảm, điều chỉnh", "(19) Ghi chú",
}

// indicator is one line of a declaration form: its number on the form, label and code.
type indicator struct{ STT, Label, Code string }

const insurancePremiumLabel = "tiền phí mua bảo hiểm nhân thọ, bảo hiểm không bắt buộc khác của doanh nghiệp bảo hiểm không thành lập tại Việt Nam cho người lao động"

// layout05KK_TT89_2026 is 05/KK-TNCN of Thông tư 89/2026/TT-BTC.
var layout05KK_TT89_2026 = []indicator{
	{"1", "Tổng số người lao động:", "[15]"},
	{"", "Trong đó: Cá nhân cư trú có hợp đồng lao động", "[16]"},
	{"2", "Tổng số cá nhân đã khấu trừ thuế [17]=[18]+[19]", "[17]"},
	{"2.1", "Cá nhân cư trú", "[18]"},
	{"2.2", "Cá nhân không cư trú", "[19]"},
	{"3", "Tổng thu nhập chịu thuế trả cho cá nhân [20]=[21]+[22]", "[20]"},
	{"3.1", "Cá nhân cư trú", "[21]"},
	{"3.2", "Cá nhân không cư trú", "[22]"},
	{"3.3", "Trong đó: Tổng thu nhập chịu thuế từ " + insurancePremiumLabel, "[23]"},
	{"4", "Tổng thu nhập được miễn trong kỳ [24]=[25]+[26]+[27]+[28]", "[24]"},
	{"4.1", "Theo quy định của Hợp đồng dầu khí", "[25]"},
	{"4.2", "Từ thực hiện nhiệm vụ khoa học, công nghệ và đổi mới sáng tạo.", "[26]"},
	{"4.3", "Thu nhập của chuyên gia hỗ trợ khởi nghiệp sáng tạo theo quy định", "[27]"},
	{"4.4", "Thu nhập được miễn khác", "[28]"},
	{"5", "Tổng thu nhập chịu thuế trả cho cá nhân thuộc diện phải khấu trừ thuế [29]=[30]+[31]", "[29]"},
	{"5.1", "Cá nhân cư trú", "[30]"},
	{"5.2", "Cá nhân không cư trú", "[31]"},
	{"6", "Thu nhập tính thuế [32]=[33]+[34]", "[32]"},
	{"6.1", "Cá nhân cư trú", "[33]"},
	{"6.2", "Cá nhân không cư trú", "[34]"},
	{"7", "Số thuế thu nhập cá nhân phải khấu trừ trong kỳ", "[35]"},
	{"8", "Số thuế được miễn trong kỳ", "[36]"},
	{"8.1", "Tổng thu nhập của cá nhân là nhân lực công nghiệp công nghệ số chất lượng cao thuộc trường hợp được miễn thuế", "[37]"},
	{"8.2", "Số thuế được miễn của cá nhân là nhân lực công nghiệp công nghệ số chất lượng cao", "[38]"},
	{"8.3", "Tổng thu nhập của cá nhân là nhân lực công nghệ cao thuộc trường hợp được miễn thuế", "[39]"},
	{"8.4", "Số thuế được miễn của cá nhân là nhân lực công nghệ cao thực hiện hoạt động nghiên cứu và phát triển công nghệ cao hoặc công nghệ chiến lược", "[40]"},
	{"8.5", "Số thuế được miễn khác theo quy định", "[41]"},
	{"9", "Tổng số thuế thu nhập cá nhân đã khấu trừ [42]=[43]+[44]", "[42]"},
	{"9.1", "Cá nhân cư trú", "[43]"},
	{"9.2", "Cá nhân không cư trú", "[44]"},
	{"9.3", "Trong đó: Tổng số thuế thu nhập cá nhân đã khấu trừ trên " + insurancePremiumLabel, "[45]"},
}

// layout05_1BK_TT89_2026 is table 1 of 05-1/BK-QTT-TNCN of Thông tư 89/2026/TT-BTC.
var layout05_1BK_TT89_2026 = []string{
	"[06] STT", "[07] Họ và tên", "[08] Mã số thuế", "[09] Số ĐDCN/ Hộ chiếu (trường hợp chưa có MST)",
	"[10] Cá nhân uỷ quyền quyết toán thay", "[11] CN nước ngoài ủy quyền quyết toán dưới 12 tháng",
	"[12] Tổng thu nhập chịu thuế phát sinh trong kỳ", "[13] Trong đó: thu nhập chịu thuế được miễn theo Hiệp định",
	"[14] Trong đó: thu nhập chịu thuế được miễn theo quy định của Hợp đồng dầu khí", "[15] Số lượng NPT tính giảm trừ",
	"[16] Tổng số tiền giảm trừ gia cảnh", "[17] Từ thiện, nhân đạo, khuyến học", "[18] Bảo hiểm được trừ",
	"[19] Quĩ hưu trí tự nguyện được trừ", "[20] Thu nhập tính thuế", "[21] Tổng số thuế TNCN đã khấu trừ",
	"[22] Tổng số thuế phải nộp", "[23] Số thuế đã nộp thừa", "[24] Số thuế còn phải nộp",
	"[25] Cá nhân có số thuế được miễn do có số thuế còn phải nộp từ 50.000 đồng trở xuống",
}

// layout05QTT_TT89_2026 is 05/QTT-TNCN of Thông tư 89/2026/TT-BTC; the form's numbering skips 7.
var layout05QTT_TT89_2026 = []indicator{
	{"I", "NGHĨA VỤ KHẤU TRỪ THUẾ CỦA TỔ CHỨC, CÁ NHÂN TRẢ THU NHẬP", ""},
	{"1", "Tổng số người lao động:", "[15]"},
	{"", "Trong đó: Cá nhân cư trú có hợp đồng lao động", "[16]"},
	{"2", "Tổng số cá nhân đã khấu trừ thuế [17]=[18]+[19]", "[17]"},
	{"2.1", "Cá nhân cư trú", "[18]"},
	{"2.2", "Cá nhân không cư trú", "[19]"},
	{"3", "Tổng số cá nhân thuộc diện được miễn, giảm thuế theo Hiệp định tránh đánh thuế hai lần", "[20]"},
	{"4", "Tổng số cá nhân giảm trừ gia cảnh", "[21]"},
	{"5", "Tổng thu nhập chịu thuế trả cho cá nhân [22]=[23]+[24]", "[22]"},
	{"5.1", "Cá nhân cư trú", "[23]"},
	{"5.2", "Cá nhân không cư trú", "[24]"},
	{"5.3", "Trong đó: Tổng thu nhập chịu thuế từ " + insurancePremiumLabel, "[25]"},
	{"6", "Tổng thu nhập được miễn trong kỳ [26]= [27]+[28]+[29]", "[26]"},
	{"6.1", "Theo quy định của Hợp đồng dầu khí", "[27]"},
	{"6.2", "Từ thực hiện nhiệm vụ khoa học, công nghệ và đổi mới sáng tạo", "[28]"},
	{"6.3", "Thu nhập của chuyên gia hỗ trợ khởi nghiệp sáng tạo", "[29]"},
	{"8", "Tổng thu nhập chịu thuế trả cho cá nhân thuộc diện phải khấu trừ thuế [30]=[31]+[32]", "[30]"},
	{"8.1", "Cá nhân cư trú", "[31]"},
	{"8.2", "Cá nhân không cư trú", "[32]"},
	{"9", "Tổng số thuế thu nhập cá nhân đã khấu trừ [33]=[34]+[35]", "[33]"},
	{"9.1", "Cá nhân cư trú", "[34]"},
	{"9.2", "Cá nhân không cư trú", "[35]"},
	{"9.3", "Trong đó: Tổng số thuế thu nhập cá nhân đã khấu trừ trên " + insurancePremiumLabel, "[36]"},
	{"II", "NGHĨA VỤ QUYẾT TOÁN THAY CHO CÁ NHÂN", ""},
	{"1", "Tổng số cá nhân uỷ quyền cho tổ chức, cá nhân trả thu nhập quyết toán thay", "[37]"},
	{"2", "Tổng số thuế thu nhập cá nhân đã khấu trừ", "[38]"},
	{"", "Trong đó: Số thuế thu nhập cá nhân đã khấu trừ tại tổ chức trước khi điều chuyển (trường hợp có đánh dấu vào chỉ tiêu [04])", "[39]"},
	{"3", "Tổng số thuế thu nhập cá nhân phải nộp", "[40]"},
	{"4", "Tổng số thuế thu nhập cá nhân được miễn do cá nhân có số thuế còn phải nộp sau ủy quyền quyết toán từ 50.000 đồng trở xuống", "[41]"},
	{"5", "Tổng số thuế thu nhập cá nhân được miễn của cá nhân là nhân lực công nghiệp công nghệ số chất lượng cao", "[42]"},
	{"6", "Tổng số thuế thu nhập cá nhân được miễn của cá nhân là nhân lực công nghệ cao", "[43]"},
	{"7", "Tổng số thuế thu nhập cá nhân còn phải nộp [44] = ([40] - [38] - [41]-[42]-[43]) >0", "[44]"},
	{"8", "Tổng số thuế thu nhập cá nhân đã nộp thừa [45] = ([40] - [38] - [41]-[42]-[43]) <0", "[45]"},
}

// layout05_3BK_TT89_2026 is 05-3/BK-QTT-TNCN of Thông tư 89/2026/TT-BTC.
var layout05_3BK_TT89_2026 = []string{
	"[06] STT", "[07] Họ và tên người nộp thuế là người lao động tại tổ chức trả thu nhập",
	"[08] MST của người nộp thuế là người lao động tại tổ chức trả thu nhập", "[09] Họ và tên người phụ thuộc",
	"[10] Ngày sinh người phụ thuộc", "[11] MST của người phụ thuộc", "[12] Loại giấy tờ (Số ĐDCN/Hộ chiếu/GKS) người phụ thuộc",
	"[13] Số giấy tờ", "[14] Quan hệ với người nộp thuế", "[15] Từ tháng", "[16] Đến tháng",
}

// relationships05_3BK are the form's words for a dependant's relationship.
var relationships05_3BK = map[string]string{"child": "Con", "spouse": "Vợ/chồng", "parent": "Cha/mẹ", "other": "Khác"}
