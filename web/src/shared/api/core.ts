// The core API as areas see it; areas import nothing else from shared/api.
import type { components } from './schema.gen'

export { coreClient as api, unwrap } from './client'
export { ApiError } from './error'

export type Me = components['schemas']['Me']
export type OrgUnit = components['schemas']['OrgUnit']
export type OrgUnitInput = components['schemas']['OrgUnitInput']
export type User = components['schemas']['User']
export type Role = components['schemas']['Role']
export type Grant = components['schemas']['Grant']
export type InboxItem = components['schemas']['InboxItem']
export type ApprovalInstance = components['schemas']['Instance']
export type ApprovalStep = components['schemas']['StepView']
export type HistoryEntry = components['schemas']['HistoryEntry']
export type Attachment = components['schemas']['Attachment']
export type AttachmentList = components['schemas']['AttachmentList']
export type Discussion = components['schemas']['Discussion']
export type Notification = components['schemas']['Notification']
export type MailServer = components['schemas']['MailServer']
export type PrintTemplate = components['schemas']['PrintTemplate']
export type PrintTemplateBlocks = components['schemas']['PrintTemplateBlocks']
export type PeriodLock = components['schemas']['PeriodLock']
export type LegalEntitySetting = components['schemas']['LegalEntitySetting']
export type TypeRule = components['schemas']['TypeRule']
export type RuleInput = components['schemas']['RuleInput']
export type RuleStep = components['schemas']['Step']
export type RuleCondition = components['schemas']['Condition']
