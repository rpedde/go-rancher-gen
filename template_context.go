package main

import (
	"fmt"
	"regexp"
	"strings"
)

type NotFoundError struct {
	msg string
}

func (e NotFoundError) Error() string {
	return e.msg
}

type TemplateContext struct {
	Services   []Service
	Containers []Container
	Hosts      []Host
	Self       Self
}

// GetContainer returns the container matching the given name.
func (c *TemplateContext) GetContainer(v ...string) (Container, error) {
	container_name := ""
	if len(v) > 0 {
		container_name = v[0]
	}
	if container_name == "" {
		container_name = c.Self.ContainerName
	}

	for _, container := range c.Containers {
		if strings.EqualFold(container_name, container.Name) {
			return container, nil
		}
	}

	return Container{}, NotFoundError{"(container) could not find host by name: " + container_name}
}

// GetHost returns the Host with the given UUID. If the argument is omitted
// the local host is returned.
func (c *TemplateContext) GetHost(v ...string) (Host, error) {
	uuid := ""
	if len(v) > 0 {
		uuid = v[0]
	}
	if uuid == "" {
		uuid = c.Self.HostUUID
	}

	for _, h := range c.Hosts {
		if strings.EqualFold(uuid, h.UUID) {
			return h, nil
		}
	}

	return Host{}, NotFoundError{"(host) could not find host by UUID: " + uuid}
}

// GetService returns the service matching the given name.
// It expects a string in the form 'service-name[.stack-name]'.
// If the argument is an empty string it returns the service of the current container.
func (c *TemplateContext) GetService(v ...string) (Service, error) {
	identifier := ""
	if len(v) > 0 {
		identifier = v[0]
	}
	var stack, service string
	if identifier == "" {
		stack = c.Self.Stack
		service = c.Self.Service
	} else {
		parts := strings.Split(identifier, ".")
		switch len(parts) {
		case 1:
			service = parts[0]
			stack = c.Self.Stack
		case 2:
			service = parts[0]
			stack = parts[1]
		default:
			return Service{}, fmt.Errorf("(service) invalid service identifier '%s'", identifier)
		}
	}

	for _, s := range c.Services {
		if strings.EqualFold(s.Name, service) && strings.EqualFold(s.Stack, stack) {
			return s, nil
		}
	}

	return Service{}, NotFoundError{"(service) could not find service by identifier: " + identifier}
}

func (c *TemplateContext) GetContainers(selectors ...string) ([]Container, error) {
	if len(selectors) == 0 {
		return c.Containers, nil
	}

	labels := LabelMap{}

	for _, f := range selectors {
		if !strings.HasPrefix(f, "@") {
			return nil, fmt.Errorf("(containers) invalid argument '%s'", f)
		}
		f = f[1:len(f)]
		parts := strings.Split(f, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("(containers) malformed label selector '%s'", f)
		}
		labels[parts[0]] = parts[1]
	}

	return filterContainersByLabel(c.Containers, labels), nil
}

func (c *TemplateContext) GetHosts(selectors ...string) ([]Host, error) {
	if len(selectors) == 0 {
		return c.Hosts, nil
	}

	labels := LabelMap{}

	for _, f := range selectors {
		if !strings.HasPrefix(f, "@") {
			return nil, fmt.Errorf("(hosts) invalid argument '%s'", f)
		}
		f = f[1:len(f)]
		parts := strings.Split(f, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("(hosts) malformed label selector '%s'", f)
		}
		labels[parts[0]] = parts[1]
	}

	return filterHostsByLabel(c.Hosts, labels), nil
}

func (c *TemplateContext) GetServices(selectors ...string) ([]Service, error) {
	if len(selectors) == 0 {
		return c.Services, nil
	}

	labels := LabelMap{}
	var stack string

	for _, f := range selectors {
		switch f[:1] {
		case ".":
			if len(stack) > 0 {
				return nil, fmt.Errorf("(services) invalid use of multiple stack selectors '%s'", f)
			}
			stack = f[1:len(f)]
		case "@":
			parts := strings.Split(f[1:len(f)], "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("(services) malformed label selector '%s'", f)
			}
			labels[parts[0]] = parts[1]
		default:
			return nil, fmt.Errorf("(services) invalid argument '%s'", f)
		}
	}

	services := c.Services

	if len(stack) > 0 {
		services = filterServicesByStack(services, stack)
	}
	if len(labels) > 0 {
		services = filterServicesByLabel(services, labels)
	}

	return services, nil
}

// returns true if the LabelMap needle is a subset of the LabelMap stack.
// the needle map may contain regex in it's values.
func inLabelMap(stack, needle LabelMap) bool {
	match := true
	for k, v := range needle {
		if stack.Exists(k) {
			if strings.EqualFold(stack.GetValue(k), v) {
				continue
			}
			// regex match
			rx, err := regexp.Compile(v)
			if err == nil && rx.MatchString(stack.GetValue(k)) {
				continue
			}
		}
		match = false
		break
	}
	return match
}

func filterContainersByLabel(containers []Container, labels LabelMap) []Container {
	result := make([]Container, 0)
	for _, c := range containers {
		if ok := inLabelMap(c.Labels, labels); ok {
			result = append(result, c)
		}
	}
	return result
}

func filterHostsByLabel(hosts []Host, labels LabelMap) []Host {
	result := make([]Host, 0)
	for _, h := range hosts {
		if ok := inLabelMap(h.Labels, labels); ok {
			result = append(result, h)
		}
	}
	return result
}

func filterServicesByLabel(services []Service, labels LabelMap) []Service {
	result := make([]Service, 0)
	for _, s := range services {
		if ok := inLabelMap(s.Labels, labels); ok {
			result = append(result, s)
		}
	}
	return result
}

func filterServicesByStack(services []Service, stack string) []Service {
	result := make([]Service, 0)
	for _, s := range services {
		if strings.EqualFold(s.Stack, stack) {
			result = append(result, s)
		}
	}
	return result
}
