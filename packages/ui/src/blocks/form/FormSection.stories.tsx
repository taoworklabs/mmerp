import type { Meta, StoryObj } from '@storybook/react-vite'
import { IconUser } from '@tabler/icons-react'
import { expect } from 'storybook/test'
import { Badge } from '../../primitives/feedback/Badge.tsx'
import { formatDate, monthLabels, parseDate, viLocale } from '../../primitives/dates/storySamples.ts'
import { DateInput } from '../../primitives/dates/DateInput.tsx'
import { Select, type SelectItem } from '../../primitives/input/Select.tsx'
import { TextInput } from '../../primitives/input/TextInput.tsx'
import { Textarea } from '../../primitives/input/Textarea.tsx'
import { FormSection } from './FormSection.tsx'

const meta: Meta<typeof FormSection> = {
  title: 'Blocks/FormSection',
  component: FormSection,
}

export default meta

type Story = StoryObj<typeof FormSection>

const orgUnits: SelectItem[] = [
  { value: 'sales', label: 'Kinh doanh' },
  { value: 'hr', label: 'Nhân sự' },
]

const managers: SelectItem[] = [
  { value: 'a', label: 'Nguyễn Văn A' },
  { value: 'b', label: 'Trần Thị B' },
]

export const Side: Story = {
  args: {
    title: 'Thông tin chính',
    children: (
      <>
        <FormSection.Row>
          <TextInput label="Họ và tên" />
        </FormSection.Row>
        <TextInput label="Mã nhân viên" />
        <Select label="Đơn vị" data={orgUnits} value={null} onChange={() => {}} placeholder="Chọn đơn vị" />
        <DateInput
          label="Ngày vào làm"
          value={null}
          onChange={() => {}}
          parse={parseDate}
          format={formatDate}
          locale={viLocale}
          labels={monthLabels}
          placeholder="dd/mm/yyyy"
        />
        <Select label="Quản lý trực tiếp" data={managers} value={null} onChange={() => {}} placeholder="Chọn quản lý" />
        <TextInput label="Tên đăng nhập" />
      </>
    ),
  },
  play: async ({ canvasElement }) => {
    const controls = canvasElement.querySelectorAll('[data-mmerp-control]')
    expect(controls.length).toBeGreaterThan(0)
    for (const control of controls) {
      expect(control.getBoundingClientRect().height).toBe(40)
    }
  },
}

export const Compact: Story = {
  args: {
    title: 'Chi tiết yêu cầu',
    layout: 'compact',
    children: (
      <>
        <TextInput label="Loại nghỉ" />
        <DateInput
          label="Từ ngày"
          value={null}
          onChange={() => {}}
          parse={parseDate}
          format={formatDate}
          locale={viLocale}
          labels={monthLabels}
          placeholder="dd/mm/yyyy"
        />
        <DateInput
          label="Đến ngày"
          value={null}
          onChange={() => {}}
          parse={parseDate}
          format={formatDate}
          locale={viLocale}
          labels={monthLabels}
          placeholder="dd/mm/yyyy"
        />
        <TextInput label="Số ngày" />
      </>
    ),
  },
}

export const Wide: Story = {
  args: {
    title: 'Diễn giải',
    layout: 'wide',
    children: <Textarea label="Lý do" />,
  },
}

export const WithBadge: Story = {
  args: {
    title: 'Thông tin chính',
    icon: <IconUser size={16} stroke={1.5} aria-hidden />,
    badge: (
      <Badge color="danger" aria-label="2 trường còn lỗi">
        2
      </Badge>
    ),
    children: (
      <>
        <TextInput label="Họ và tên" error="Bắt buộc" />
        <TextInput label="Mã nhân viên" error="Bắt buộc" />
      </>
    ),
  },
}
