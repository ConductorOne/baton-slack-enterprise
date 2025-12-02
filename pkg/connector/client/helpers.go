package enterprise

func logBody(body []byte, size int) string {
	if len(body) > size {
		return string(body[:size]) + " ..."
	}
	return string(body)
}
