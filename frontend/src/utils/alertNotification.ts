import type { Severity } from '@/types/api'

// 音频通知管理类
class AlertNotificationManager {
  private audioContext: AudioContext | null = null
  private enabled: boolean = false
  private lastPlayTime: number = 0
  private minInterval: number = 3000 // 最小播放间隔（毫秒）

  constructor() {
    // 检查浏览器支持
    if (typeof window !== 'undefined' && 'AudioContext' in window) {
      this.audioContext = new AudioContext()
    }
  }

  // 启用音频通知
  enable() {
    this.enabled = true
    // 恢复AudioContext（用户手势后才能播放）
    if (this.audioContext && this.audioContext.state === 'suspended') {
      this.audioContext.resume()
    }
  }

  // 禁用音频通知
  disable() {
    this.enabled = false
  }

  // 检查是否启用
  isEnabled(): boolean {
    return this.enabled
  }

  // 播放告警声音
  async play(severity: Severity) {
    if (!this.enabled || !this.audioContext) {
      return
    }

    // 防抖：避免短时间内重复播放
    const now = Date.now()
    if (now - this.lastPlayTime < this.minInterval) {
      return
    }
    this.lastPlayTime = now

    try {
      // 根据严重程度生成不同频率的声音
      const frequencies = this.getFrequencies(severity)
      const duration = this.getDuration(severity)

      for (let i = 0; i < frequencies.length; i++) {
        await this.playTone(frequencies[i], duration / frequencies.length, i * (duration / frequencies.length))
      }
    } catch (error) {
      console.error('Failed to play alert sound:', error)
    }
  }

  // 根据严重程度获取频率序列
  private getFrequencies(severity: Severity): number[] {
    switch (severity) {
      case 'critical':
        // 高频急促警报
        return [880, 988, 880, 988, 880]
      case 'high':
        // 中高频警报
        return [660, 784, 660]
      case 'medium':
        // 中频提示音
        return [523, 659]
      case 'low':
        // 低频单音
        return [440]
      default:
        return [440]
    }
  }

  // 根据严重程度获取持续时间
  private getDuration(severity: Severity): number {
    switch (severity) {
      case 'critical':
        return 1.5 // 秒
      case 'high':
        return 1.0
      case 'medium':
        return 0.6
      case 'low':
        return 0.3
      default:
        return 0.3
    }
  }

  // 播放单个音调
  private playTone(frequency: number, duration: number, delay: number = 0): Promise<void> {
    return new Promise((resolve) => {
      if (!this.audioContext) {
        resolve()
        return
      }

      setTimeout(() => {
        if (!this.audioContext) {
          resolve()
          return
        }

        const oscillator = this.audioContext.createOscillator()
        const gainNode = this.audioContext.createGain()

        oscillator.connect(gainNode)
        gainNode.connect(this.audioContext.destination)

        oscillator.frequency.value = frequency
        oscillator.type = 'sine'

        // 音量包络（淡入淡出）
        const now = this.audioContext.currentTime
        gainNode.gain.setValueAtTime(0, now)
        gainNode.gain.linearRampToValueAtTime(0.3, now + 0.01)
        gainNode.gain.linearRampToValueAtTime(0.3, now + duration - 0.05)
        gainNode.gain.linearRampToValueAtTime(0, now + duration)

        oscillator.start(now)
        oscillator.stop(now + duration)

        oscillator.onended = () => {
          resolve()
        }
      }, delay * 1000)
    })
  }

  // 测试声音
  async test() {
    const previousState = this.enabled
    this.enabled = true
    await this.play('medium')
    this.enabled = previousState
  }

  // 清理资源
  dispose() {
    if (this.audioContext) {
      this.audioContext.close()
      this.audioContext = null
    }
  }
}

// 导出单例实例
export const alertNotification = new AlertNotificationManager()

// 从localStorage读取用户设置
export const initAlertNotification = () => {
  const enabled = localStorage.getItem('alert_notification_enabled')
  if (enabled === 'true') {
    alertNotification.enable()
  }
}

// 保存用户设置
export const setAlertNotificationEnabled = (enabled: boolean) => {
  localStorage.setItem('alert_notification_enabled', enabled.toString())
  if (enabled) {
    alertNotification.enable()
  } else {
    alertNotification.disable()
  }
}
