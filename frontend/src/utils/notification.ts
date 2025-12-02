import { notificationAPI } from '../composables/useNotification'

/**
 * 全局通知API
 * 可以在应用的任何地方导入并使用
 * 
 * @example
 * ```ts
 * import { notification } from '@/utils/notification'
 * 
 * // 显示成功通知（3秒后自动消失）
 * notification.success('操作成功！')
 * 
 * // 显示持久通知
 * const id = notification.info('正在处理...', { persistent: true })
 * 
 * // 更新通知
 * notification.update(id, { 
 *   type: 'success', 
 *   message: '处理完成！',
 *   persistent: false 
 * })
 * 
 * // 带按钮的通知
 * notification.warning('发现新版本', {
 *   button: {
 *     text: '更新',
 *     onClick: () => console.log('开始更新')
 *   }
 * })
 * ```
 */
export const notification = notificationAPI

// 也可以导出为默认值
export default notification

