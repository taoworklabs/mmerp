import type { Meta, StoryObj } from '@storybook/react-vite'
import { Grid } from './Grid.tsx'

const meta: Meta<typeof Grid> = {
  title: 'Layout/Grid',
  component: Grid,
}

export default meta

type Story = StoryObj<typeof Grid>

export const Default: Story = {
  render: () => (
    <Grid>
      <Grid.Col span={6}>Cột trái</Grid.Col>
      <Grid.Col span={6}>Cột phải</Grid.Col>
    </Grid>
  ),
}

export const Columns: Story = {
  render: () => (
    <Grid columns={4}>
      <Grid.Col span={1}>1</Grid.Col>
      <Grid.Col span={1}>2</Grid.Col>
      <Grid.Col span={1}>3</Grid.Col>
      <Grid.Col span={1}>4</Grid.Col>
    </Grid>
  ),
}

export const Gap: Story = {
  render: () => (
    <Grid gap="xl">
      <Grid.Col span={6}>Cột trái</Grid.Col>
      <Grid.Col span={6}>Cột phải</Grid.Col>
    </Grid>
  ),
}

export const ColSpanAuto: Story = {
  render: () => (
    <Grid>
      <Grid.Col span="auto">Tự động</Grid.Col>
      <Grid.Col span={6}>Cố định</Grid.Col>
    </Grid>
  ),
}

export const Responsive: Story = {
  render: () => (
    <Grid>
      <Grid.Col span={{ base: 12, md: 4 }} data-testid="c">
        Co giãn theo màn hình
      </Grid.Col>
    </Grid>
  ),
}
