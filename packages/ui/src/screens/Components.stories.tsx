import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, within } from 'storybook/test'
import {
  Alert, Badge, Button, Checkbox, FileInput, FormFields, Group, MultiSelect,
  NumberInput, Page, PageHeader, Panel, PasswordInput, Select, Stack, Text, Textarea, TextInput,
} from '../index.ts'

const options = [{ value: 'a', label: 'Nguyễn Văn An' }, { value: 'b', label: 'Đặng Thị Bích' }]

function Components() {
  const [selected, setSelected] = useState<string | null>(null)
  const [multiple, setMultiple] = useState<string[]>(['a', 'b'])
  const [quantity, setQuantity] = useState<number | null>(12)
  const [file, setFile] = useState<File | null>(null)
  return (
    <Page header={<PageHeader title="Thành phần giao diện" description="Kiểu chữ, trạng thái và tương tác dùng chung" />}>
      <Stack gap="lg">
        <Panel title="Thao tác và trạng thái">
          <Stack>
            <Group>
              <Button>Lưu thay đổi</Button><Button variant="default">Xuất dữ liệu</Button>
              <Button variant="subtle" color="gray">Hủy</Button><Button loading>Đang lưu</Button><Button disabled>Không khả dụng</Button>
            </Group>
            <Group>
              <Badge color="gray">Nháp</Badge><Badge color="warning">Chờ duyệt</Badge>
              <Badge color="success">Đã duyệt</Badge><Badge color="danger">Đã hủy</Badge><Badge color="info">Thông tin</Badge>
            </Group>
            <Alert color="info" title="Thông tin">Các thay đổi sẽ được lưu sau khi bạn xác nhận.</Alert>
          </Stack>
        </Panel>
        <Panel title="Trường nhập liệu">
          <FormFields>
            <TextInput label="Họ và tên" placeholder="Nhập họ và tên" required />
            <TextInput label="Mã nhân viên" defaultValue="NV00001" readOnly description="Giá trị chỉ đọc vẫn có thể chọn và sao chép" />
            <TextInput label="Email" error="Nhập địa chỉ email hợp lệ" defaultValue="an@" />
            <TextInput label="Trường không khả dụng" disabled placeholder="Không thể chỉnh sửa" />
            <PasswordInput label="Mật khẩu" visibilityLabels={{ show: 'Hiện mật khẩu', hide: 'Ẩn mật khẩu' }} />
            <NumberInput label="Số lượng" value={quantity} onChange={setQuantity} />
            <Select label="Người phụ trách" data={options} value={selected} onChange={setSelected} placeholder="Chọn người phụ trách" clearable clearLabel="Xóa lựa chọn" />
            <MultiSelect label="Người duyệt" data={options} value={multiple} onChange={setMultiple} nothingFound="Không tìm thấy" removeLabel={name => `Bỏ chọn ${name}`} />
            <FileInput label="Tệp đính kèm" value={file} onChange={setFile} placeholder="Chọn tệp" clearLabel="Xóa tệp" />
            <Textarea label="Ghi chú" placeholder="Thêm ghi chú" />
          </FormFields>
        </Panel>
        <Panel title="Lựa chọn">
          <Stack gap="xs">
            <Checkbox label="Gửi thông báo khi hoàn tất" defaultChecked />
            <Checkbox label="Tôi đồng ý với điều khoản" error="Bạn cần xác nhận trước khi tiếp tục" />
            <Checkbox label="Thiết lập do quản trị viên quản lý" disabled />
            <Text size="xs" tone="dimmed">Dùng Tab để di chuyển, Space để chọn và Escape để đóng danh sách.</Text>
          </Stack>
        </Panel>
      </Stack>
    </Page>
  )
}

const meta: Meta<typeof Components> = { title: 'Foundations/Components', component: Components, parameters: { layout: 'fullscreen' } }
export default meta
type Story = StoryObj<typeof Components>
export const Desktop: Story = {}
export const Phone: Story = {
  globals: { viewport: { value: 'mobile' } },
  play: async ({ canvasElement }) => {
    const root = canvasElement.ownerDocument.documentElement
    expect(root.scrollWidth).toBeLessThanOrEqual(root.clientWidth)
    expect(within(canvasElement).getByRole('button', { name: /Tệp đính kèm/ })).toBeInTheDocument()
  },
}
