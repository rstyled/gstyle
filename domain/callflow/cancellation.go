package callflow

func (body *exitBody) canceled() bool {
	return body != nil && body.ctx != nil && body.ctx.Err() != nil
}
