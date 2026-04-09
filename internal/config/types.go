package config

import "time"

// Config is the root configuration for moor.
type Config struct {
	Settings     Settings      `yaml:"settings"`
	Reservations []Reservation `yaml:"reservations,omitempty"`
	ForwardRules []ForwardRule `yaml:"forward_rules,omitempty"`
}

// Settings holds global moor preferences.
type Settings struct {
	RefreshIntervalSecs int    `yaml:"refresh_interval_seconds"`
	GracePeriodSecs     int    `yaml:"grace_period_seconds"`
	ShowDocker          bool   `yaml:"show_docker"`
	DefaultOutput       string `yaml:"default_output"`
}

// Reservation binds a port to a project.
type Reservation struct {
	Port        uint16 `yaml:"port"`
	Project     string `yaml:"project"`
	Description string `yaml:"description,omitempty"`
	EnvFile     string `yaml:"env_file,omitempty"`
	CreatedAt   string `yaml:"created_at,omitempty"`
}

// ForwardRule defines a persistent port forwarding rule.
type ForwardRule struct {
	Name     string `yaml:"name"`
	FromPort uint16 `yaml:"from_port"`
	ToPort   uint16 `yaml:"to_port"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Settings: Settings{
			RefreshIntervalSecs: 2,
			GracePeriodSecs:     3,
			ShowDocker:          true,
			DefaultOutput:       "table",
		},
	}
}

// FindReservation returns the reservation for a port, or nil.
func (c Config) FindReservation(port uint16) *Reservation {
	for i := range c.Reservations {
		if c.Reservations[i].Port == port {
			return &c.Reservations[i]
		}
	}
	return nil
}

// AddReservation adds or updates a reservation. Returns a new Config.
func (c Config) AddReservation(r Reservation) Config {
	if r.CreatedAt == "" {
		r.CreatedAt = time.Now().Format(time.RFC3339)
	}

	newReservations := make([]Reservation, 0, len(c.Reservations)+1)
	replaced := false
	for _, existing := range c.Reservations {
		if existing.Port == r.Port {
			newReservations = append(newReservations, r)
			replaced = true
		} else {
			newReservations = append(newReservations, existing)
		}
	}
	if !replaced {
		newReservations = append(newReservations, r)
	}

	return Config{
		Settings:     c.Settings,
		Reservations: newReservations,
		ForwardRules: c.ForwardRules,
	}
}

// RemoveReservation removes a reservation by port. Returns a new Config.
func (c Config) RemoveReservation(port uint16) Config {
	newReservations := make([]Reservation, 0, len(c.Reservations))
	for _, r := range c.Reservations {
		if r.Port != port {
			newReservations = append(newReservations, r)
		}
	}
	return Config{
		Settings:     c.Settings,
		Reservations: newReservations,
		ForwardRules: c.ForwardRules,
	}
}
