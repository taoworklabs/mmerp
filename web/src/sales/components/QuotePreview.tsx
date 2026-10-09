import { DocPreview } from './DocPreview'

export default function QuotePreview({ id }: { id: number }) {
  return <DocPreview kind="quote" id={id} />
}
