package options

// HookOptions represents the options for configuring how hook runs on a single operation.
type HookOptions struct {
	// DisableAllHooks disables all hooks, ignoring other hook options.
	DisableAllHooks *bool

	// DisableBeforeHooks disables all before hooks.
	DisableBeforeHooks *bool

	// DisableAfterHooks disables all after hooks.
	DisableAfterHooks *bool

	// This stores the names of the hooks that are disabled.
	// Example: []string{"BeforeCreate", "AfterCreate"}
	DisabledHooks *[]string
}

// Hook returns a new HookOptions instance.
func Hook() *HookOptions {
	return &HookOptions{}
}

// SetDisableAllHooks sets the `DisableAllHooks` option.
func (opts *HookOptions) SetDisableAllHooks(disable bool) *HookOptions {
	opts.DisableAllHooks = &disable
	return opts
}

// SetDisableBeforeHooks sets the `DisableBeforeHooks` option.
func (opts *HookOptions) SetDisableBeforeHooks(disable bool) *HookOptions {
	opts.DisableBeforeHooks = &disable
	return opts
}

// SetDisableAfterHooks sets the `DisableAfterHooks` option.
func (opts *HookOptions) SetDisableAfterHooks(disable bool) *HookOptions {
	opts.DisableAfterHooks = &disable
	return opts
}

// SetDisabledHooks sets the `DisabledHooks` option.
func (opts *HookOptions) SetDisabledHooks(hooks ...string) *HookOptions {
	opts.DisabledHooks = &hooks
	return opts
}
