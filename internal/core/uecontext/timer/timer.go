package timer

import "time"

type TimerType int

const (
	T3346 TimerType = iota + 1
	T3396
	T3445
	T3502
	T3510
	T3511
	T3512
	T3516
	T3517
	T3519
	T3520
	T3521
	T3525
	T3540
	T3527
)

const (
	T3346_duration = 15 * time.Minute
	T3396_duration = 30 * time.Minute
	T3445_duration = 12 * time.Hour
	T3502_duration = 12 * time.Minute
	T3510_duration = 15 * time.Second
	T3511_duration = 10 * time.Second
	// T3502_duration = 2 * time.Second
	// T3510_duration = 2 * time.Second
	// T3511_duration = 2 * time.Second
	T3512_duration = 54 * time.Minute
	T3516_duration = 30 * time.Second
	T3517_duration = 15 * time.Second
	T3519_duration = time.Minute
	T3520_duration = 15 * time.Second
	T3521_duration = 15 * time.Second
	T3525_duration = time.Minute
	T3540_duration = 10 * time.Second
	T3527_duration = 15 * time.Second
)
