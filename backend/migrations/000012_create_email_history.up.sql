-- 创建邮件发送历史表
CREATE TABLE IF NOT EXISTS email_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID REFERENCES alerts(id) ON DELETE SET NULL,
    provider VARCHAR(50) NOT NULL,
    recipients TEXT[] NOT NULL,
    subject TEXT NOT NULL,
    status VARCHAR(20) NOT NULL, -- 'queued', 'sent', 'failed', 'retrying'
    attempts INT DEFAULT 1,
    last_error TEXT,
    sent_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_email_history_alert_id ON email_history(alert_id);
CREATE INDEX idx_email_history_status ON email_history(status);
CREATE INDEX idx_email_history_created_at ON email_history(created_at DESC);
CREATE INDEX idx_email_history_provider ON email_history(provider);

-- 添加注释
COMMENT ON TABLE email_history IS '邮件发送历史记录';
COMMENT ON COLUMN email_history.id IS '邮件记录ID';
COMMENT ON COLUMN email_history.alert_id IS '关联的告警ID';
COMMENT ON COLUMN email_history.provider IS '邮件提供商(smtp/sendgrid/mailgun/ses)';
COMMENT ON COLUMN email_history.recipients IS '收件人列表';
COMMENT ON COLUMN email_history.subject IS '邮件主题';
COMMENT ON COLUMN email_history.status IS '发送状态';
COMMENT ON COLUMN email_history.attempts IS '发送尝试次数';
COMMENT ON COLUMN email_history.last_error IS '最后一次错误信息';
COMMENT ON COLUMN email_history.sent_at IS '成功发送时间';
COMMENT ON COLUMN email_history.created_at IS '创建时间';
COMMENT ON COLUMN email_history.updated_at IS '更新时间';
