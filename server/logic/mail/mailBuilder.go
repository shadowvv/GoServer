package mail

// buildMailEntityFromMail centralizes MailEntity construction from Mail.
func buildMailEntityFromMail(m *Mail) *MailEntity {
	if m == nil {
		return &MailEntity{}
	}

	entity := &MailEntity{
		MailID:       m.MailID,
		UserID:       m.UserID,
		MailType:     m.MailType,
		Title:        m.Title,
		Content:      m.Content,
		SenderID:     m.SenderID,
		SenderName:   m.SenderName,
		SenderAvatar: m.SenderAvatar,
		ServerMailID: m.ServerMailID,
		TemplateID:   m.TemplateID,
		Status:       m.Status,
		IsConvenient: m.IsConvenient,
		ExpireTime:   m.ExpireTime,
		SendTime:     m.SendTime,
		ReadTime:     m.ReadTime,
		ClaimTime:    m.ClaimTime,
	}
	_ = entity.SetItems(m.Items)
	_ = entity.SetTitleParams(m.TitleParams)
	_ = entity.SetContentParams(m.ContentParams)
	return entity
}

// buildServerMailEntityFromServerMail centralizes ServerMailEntity construction from ServerMail.
func buildServerMailEntityFromServerMail(s *ServerMail) *ServerMailEntity {
	if s == nil {
		return &ServerMailEntity{}
	}

	entity := &ServerMailEntity{
		ServerMailID: s.ServerMailID,
		MailType:     s.MailType,
		Title:        s.Title,
		Content:      s.Content,
		TemplateID:   s.TemplateID,
		ServerID:     s.ServerID,
		AllianceID:   s.AllianceID,
		SenderAvatar: s.SenderAvatar,
		IsConvenient: s.IsConvenient,
		SendTime:     s.SendTime,
		ExpireTime:   s.ExpireTime,
		Status:       s.Status,
		CreatedBy:    s.CreatedBy,
	}
	_ = entity.SetItems(s.Items)
	_ = entity.SetUnlockList(s.UnlockList)
	_ = entity.SetTitleParams(nil)
	_ = entity.SetContentParams(s.ContentParams)
	return entity
}
