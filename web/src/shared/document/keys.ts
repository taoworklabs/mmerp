// Query keys of the core document routes; areas never write these by hand.
export const documentKeys = {
  inbox: () => ['core', 'inbox'] as const,
  document: (docType: string, id: number) => ['core', 'documents', docType, id] as const,
  history: (docType: string, id: number) => ['core', 'documents', docType, id, 'history'] as const,
  approval: (docType: string, id: number) => ['core', 'documents', docType, id, 'approval'] as const,
  // Under the document's key, so every write to the record refreshes them too.
  attachments: (docType: string, id: number) => ['core', 'documents', docType, id, 'attachments'] as const,
  discussion: (docType: string, id: number) => ['core', 'documents', docType, id, 'discussion'] as const,
  mentionable: (docType: string, id: number) => ['core', 'documents', docType, id, 'mentionable'] as const,
  approvalRules: (product?: string) => (product ? (['core', 'approval-rules', product] as const) : (['core', 'approval-rules'] as const)),
  approvalRuleUsers: () => ['core', 'approval-rule-users'] as const,
  roles: () => ['core', 'roles'] as const,
}
