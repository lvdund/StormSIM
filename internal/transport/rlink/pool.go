package rlink

// const DefaultMsgQueueSize = 25
// const DefaultMsgTimeout = 3 * time.Second
//
// var RLink *LinkPool = NewLinkPool(DefaultMsgTimeout, DefaultMsgQueueSize)
//
// type LinkPool struct {
// 	connections      sync.Map // key: "ueID:gnbID", value: *Connection
// 	totalConnections int64
//
// 	mu      sync.RWMutex
// 	timeout time.Duration
// 	buffer  int
// 	closed  int32 // atomic bool
// }
//
// func NewLinkPool(timeout time.Duration, buffer int) *LinkPool {
// 	return &LinkPool{
// 		timeout: timeout,
// 		buffer:  buffer,
// 	}
// }
//
// func (cl *LinkPool) isClosed() bool {
// 	return atomic.LoadInt32(&cl.closed) == 1
// }
//
// func (cl *LinkPool) GetConnection(ueID int64, gnbID string) (*Connection, error) {
// 	if cl.isClosed() {
// 		return nil, fmt.Errorf("link pool is closed")
// 	}
//
// 	key := ConnectionKey(ueID, gnbID)
// 	if conn, ok := cl.connections.Load(key); ok {
// 		return conn.(*Connection), nil
// 	}
//
// 	return nil, fmt.Errorf("connection not found for UE %d and GNB %s", ueID, gnbID)
// }
//
// func (cl *LinkPool) CreateConnection(ueID int64, gnbID string) (*Connection, error) {
// 	if cl.isClosed() {
// 		return nil, fmt.Errorf("link pool is closed")
// 	}
//
// 	key := ConnectionKey(ueID, gnbID)
// 	if _, exists := cl.connections.Load(key); exists {
// 		return nil, fmt.Errorf(
// 			"connection already exists for UE %d and GNB %s", ueID, gnbID)
// 	}
//
// 	// Create new connection
// 	conn := NewConnection(ueID, gnbID, cl.buffer, cl.timeout)
// 	if actual, loaded := cl.connections.LoadOrStore(key, conn); loaded {
// 		conn.Close() // Someone else created it first, close ours and return theirs
// 		return actual.(*Connection), nil
// 	}
// 	atomic.AddInt64(&cl.totalConnections, 1)
//
// 	return conn, nil
// }
//
// func (cl *LinkPool) RemoveConnection(ueID int64, gnbID string) error {
// 	key := ConnectionKey(ueID, gnbID)
//
// 	if conn, ok := cl.connections.LoadAndDelete(key); ok {
// 		conn.(*Connection).Close()
// 		return nil
// 	}
//
// 	return fmt.Errorf("connection not found")
// }
