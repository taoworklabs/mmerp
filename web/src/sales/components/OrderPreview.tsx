import { DocPreview } from './DocPreview'

export default function OrderPreview({ id }: { id: number }) {
  return <DocPreview kind="order" id={id} />
}
