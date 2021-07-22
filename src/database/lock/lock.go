package lock

// Open->Flockの順で使う、Lock獲得できてからReadとかWrite
func Flock(path string, fn func()) error {
	l := NewFileLock(path)
	l.Lock()
	defer l.Unlock()

	//lock獲得後、fdに対して操作を実行
	fn()

	return nil
}
