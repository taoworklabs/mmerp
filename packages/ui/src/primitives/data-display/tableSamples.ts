const names = [
  'Nguyễn Văn An', 'Trần Thị Bích', 'Lê Văn Cường', 'Phạm Thu Hà',
  'Hoàng Minh Đức', 'Đặng Thị Lan', 'Bùi Quốc Minh', 'Hồ Ngọc Nhi',
  'Đỗ Thanh Phong', 'Ngô Mỹ Quyên', 'Dương Tuấn Sơn', 'Lý Kim Thảo',
  'Vũ Hải Nam', 'Đinh Ngọc Anh', 'Phan Thanh Bình', 'Mai Thùy Dung',
  'Trương Quang Huy', 'Đoàn Bảo Ngọc', 'Tạ Minh Khang', 'Cao Thu Trang',
  'Võ Thành Đạt', 'Nguyễn Khánh Linh', 'Trần Quốc Hưng', 'Lê Phương Vy',
]

const departments = ['Kế toán', 'Kinh doanh', 'Nhân sự', 'Vận hành', 'Kỹ thuật', 'Chăm sóc khách hàng']
const roles = ['Kế toán viên', 'Nhân viên kinh doanh', 'Chuyên viên nhân sự', 'Điều phối viên', 'Kỹ sư phần mềm', 'Chuyên viên hỗ trợ']

// Fixed samples keep sorting, selection and scrolling reproducible across stories.
export const tableEmployees = names.map((full_name, index) => ({
  id: index + 1,
  full_name,
  code: `NV${String(index + 1).padStart(3, '0')}`,
  org_unit: departments[index % departments.length]!,
  role: roles[index % roles.length]!,
  hire_date: `${2019 + index % 7}-${String(1 + index % 12).padStart(2, '0')}-${String(1 + index % 27).padStart(2, '0')}`,
  status: index % 7 === 2 ? 'terminated' as const : 'active' as const,
}))
