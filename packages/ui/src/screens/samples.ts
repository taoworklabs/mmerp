// Sample data for the Employees screen stories: a small, deterministic Vietnamese HRM roster.
export type Employee = {
  id: number
  code: string
  name: string
  unit: string
  hire: string
  status: 'active' | 'terminated'
  manager?: string
  login?: string
  phone?: string
  email?: string
}

export const units = ['Phòng Kế toán', 'Phòng Kinh doanh', 'Phòng Nhân sự', 'Kho Bình Dương', 'Ban Giám đốc']

const names = [
  'Nguyễn Văn An',
  'Trần Thị Bình',
  'Lê Hoàng Cường',
  'Phạm Bảo Hà',
  'Võ Minh Khôi',
  'Đặng Thu Lan',
  'Bùi Quốc Minh',
  'Hồ Ngọc Nhi',
  'Đỗ Thanh Phong',
  'Ngô Mỹ Quyên',
  'Dương Tuấn Sơn',
  'Lý Kim Thảo',
]

export const employees: Employee[] = names.map((name, i) => ({
  id: i + 1,
  code: `NV${String(i + 1).padStart(4, '0')}`,
  name,
  unit: units[i % units.length] ?? '',
  hire: `20${18 + (i % 7)}-0${(i % 9) + 1}-1${i % 9}`,
  status: i % 5 === 4 ? 'terminated' : 'active',
  manager: i > 0 ? names[0] : undefined,
  login: i % 3 === 0 ? `user${i + 1}` : undefined,
  phone: i % 2 === 0 ? '0903 123 456' : undefined,
  email: i % 2 === 0 ? `nv${i + 1}@demo.vn` : undefined,
}))
