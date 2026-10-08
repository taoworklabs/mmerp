// The header's unread count and the notifications page; reading one refreshes both.
export const notificationKeys = {
  all: () => ['core', 'notifications'] as const,
  unread: () => ['core', 'notifications', 'unread'] as const,
  list: () => ['core', 'notifications', 'list'] as const,
}
