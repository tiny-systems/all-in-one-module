package common

import (
	"context"
	"fmt"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"time"
)

const (
	SchedulerComponent        = "scheduler"
	SchedulerOutPort   string = "out"
	SchedulerInPort    string = "in"
	SchedulerAckPort   string = "ack"
)

type SchedulerSettings struct {
	EnableAckPort bool `json:"enableAckPort" title:"Enable task acknowledge port" description:"Port gives information if incoming task was scheduled properly"`
}

type SchedulerContext any

type SchedulerInMessage struct {
	Context SchedulerContext `json:"context" title:"Context" configurable:"true" description:"Arbitrary message to be send further" propertyOrder:"1"`
	Task    Task             `json:"task" title:"Task" required:"true" propertyOrder:"2"`
}

type Task struct {
	ID       string    `json:"id" required:"true" title:"Unique task ID" propertyOrder:"1"`
	DateTime time.Time `json:"dateTime" required:"true" title:"Date and time" description:"Format examples: 2012-10-01T09:45:00.000+02:00" propertyOrder:"2"`
	Schedule bool      `json:"schedule" required:"true" title:"Schedule" description:"You can unschedule existing task by settings schedule equals false. Default: true" propertyOrder:"3"`
}

type SchedulerOutMessage struct {
	Task    Task             `json:"task"`
	Context SchedulerContext `json:"context"`
}

type SchedulerTaskAck struct {
	Task        Task             `json:"task"`
	Context     SchedulerContext `json:"context"`
	ScheduledIn int64            `json:"scheduledIn"`
}

type task struct {
	timer *time.Timer
	call  func()
	id    string
}

type Scheduler struct {
	settings SchedulerSettings
	tasks    cmap.ConcurrentMap[string, *task]
}

func (s *Scheduler) Instance() module.Component {
	return &Scheduler{
		tasks: cmap.New[*task](),
	}
}

func (s *Scheduler) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        SchedulerComponent,
		Description: "Scheduler",
		Info:        "Collects tasks messages. When its running sends messages further when scheduled date and time come. Tasks with same IDs are updating scheduled date and task itself. If scheduled date is already passed - sends message as soon as being started",
		Tags:        []string{"SDK"},
	}
}

func (s *Scheduler) emit(ctx context.Context) error {
	for _, k := range s.tasks.Keys() {
		v, _ := s.tasks.Get(k)
		go s.waitTask(ctx, v)
	}
	<-ctx.Done()
	return ctx.Err()
}

func (s *Scheduler) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {

	//emit
	if port == module.SettingsPort {
		in, ok := msg.(SchedulerSettings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		s.settings = in
		return nil
	}

	if port != SchedulerInPort {
		return fmt.Errorf("invalid port: %s", port)
	}

	in, ok := msg.(SchedulerInMessage)
	if !ok {
		return fmt.Errorf("invalid message")
	}

	var (
		t           = in.Task
		scheduledIn int64
	)
	if in.Task.Schedule {
		scheduledIn = int64(t.DateTime.Sub(time.Now()).Seconds())
	}

	if s.settings.EnableAckPort {
		if err := handler(ctx, SchedulerAckPort, SchedulerTaskAck{
			Task:        in.Task,
			Context:     in.Context,
			ScheduledIn: scheduledIn,
		}); err != nil {
			return err
		}
	}

	s.addOrUpdateTask(t.ID, t.Schedule, t.DateTime.Sub(time.Now()), func() {
		_ = handler(ctx, SchedulerOutPort, SchedulerOutMessage{
			Task:    in.Task,
			Context: in.Context,
		})
	})
	return nil
}

func (s *Scheduler) addOrUpdateTask(id string, start bool, duration time.Duration, f func()) {
	if d, ok := s.tasks.Get(id); ok {
		// job is registered
		// tasks it
		d.timer.Stop()
		if start {
			d.timer.Reset(duration)
		} else {
			s.tasks.Remove(id)
		}
		return
	}
	if !start {
		return
	}
	tt := &task{
		timer: time.NewTimer(duration),
		id:    id,
		call:  f,
	}
	s.tasks.Set(id, tt)
}

func (s *Scheduler) waitTask(ctx context.Context, d *task) {
	select {
	case <-d.timer.C:
		s.tasks.Remove(d.id)
		d.call()
	case <-ctx.Done():
	}
}

func (s *Scheduler) Ports() []module.NodePort {
	ports := []module.NodePort{
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
			Configuration: SchedulerSettings{},
		},
		{
			Name:   SchedulerInPort,
			Label:  "Tasks",
			Source: true,
			Configuration: SchedulerInMessage{
				Task: Task{
					ID:       "someID2323",
					DateTime: time.Now(),
					Schedule: true,
				},
			},
			Position: module.Left,
		},
		{
			Name:          SchedulerOutPort,
			Label:         "Scheduled",
			Source:        false,
			Configuration: SchedulerOutMessage{},
			Position:      module.Right,
		},
	}

	if !s.settings.EnableAckPort {
		return ports
	}

	return append(ports, module.NodePort{
		Name:          SchedulerAckPort,
		Label:         "Ack",
		Source:        false,
		Configuration: SchedulerTaskAck{},
		Position:      module.Bottom,
	})
}

var scheduler = (*Scheduler)(nil)

var _ module.Component = scheduler

func init() {
	registry.Register(&Scheduler{})
}
