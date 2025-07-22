package twitterclient

func (m *Manager) GetRateLimiter(cli *Client) *rateLimitManager {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if v, ok := m.clientRateLimiters.Load(cli); ok {
		return v
	}
	return nil
}
