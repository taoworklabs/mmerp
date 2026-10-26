import { breakpoint, useMediaQuery } from '@mmerp/ui'
import { Drawer } from '@mantine/core'
import { useTranslation } from 'react-i18next'
import { useFlowEditor } from './editor'
import { ElementPanel } from './ElementPanel'
import { FlowDiagram } from './FlowDiagram'
import c from './FlowDiagram.module.css'

// FlowWorkspace lays out the editor: the diagram, and beside it the panel of what is selected;
// on a narrow screen the diagram takes the width and the panel slides up over it.
export function FlowWorkspace() {
  const { t } = useTranslation()
  // Beside the diagram only where both fit (lg, 1280px); below that it slides up over it.
  const wide = useMediaQuery(`(min-width: ${breakpoint.lg})`, true)
  const { selected, select } = useFlowEditor()
  return (
    <div className={c.workspace}>
      <FlowDiagram />
      {wide ? (
        <aside className={c.side} aria-label={t('shared.rule.panel_title')}>
          <ElementPanel />
        </aside>
      ) : (
        <Drawer
          opened={selected !== null}
          onClose={() => select(null)}
          position="bottom"
          size="85%"
          title={t('shared.rule.panel_title')}
          closeButtonProps={{ 'aria-label': t('shared.modal.close') }}
          // Focus going back to the box would select it and open the drawer again.
          returnFocus={false}
        >
          <ElementPanel />
        </Drawer>
      )}
    </div>
  )
}
