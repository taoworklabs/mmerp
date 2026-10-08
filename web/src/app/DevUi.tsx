import { WithSampleSections } from '@/shared/document/samples'
import DevUiPage from '@/shared/ui/DevUiPage'

// DevUi is /dev/ui with the document side panels on sample data; shared/ui knows no document feature.
export default function DevUi() {
  return <DevUiPage withSections={WithSampleSections} />
}
