package emailval

// Statuses follow the ZeroBounce v2 scheme so results map one-to-one for users.
const (
	StatusValid     = "valid"
	StatusInvalid   = "invalid"
	StatusCatchAll  = "catch-all"
	StatusUnknown   = "unknown"
	StatusSpamtrap  = "spamtrap"
	StatusAbuse     = "abuse"
	StatusDoNotMail = "do_not_mail"
)

// Sub-statuses, grouped by the status they explain.
const (
	// valid
	SubAliasAddress         = "alias_address"
	SubLeadingPeriodRemoved = "leading_period_removed"
	SubAlternate            = "alternate"
	SubGold                 = "gold"
	SubRoleBasedAcceptAll   = "role_based_accept_all"
	SubAcceptAll            = "accept_all"
	SubAIAgentMailbox       = "ai_agent_mailbox" // also used under catch-all
	// invalid
	SubDoesNotAcceptMail    = "does_not_accept_mail"
	SubFailedSyntaxCheck    = "failed_syntax_check"
	SubPossibleTypo         = "possible_typo"
	SubMailboxNotFound      = "mailbox_not_found"
	SubNoDNSEntries         = "no_dns_entries"
	SubMailboxQuotaExceeded = "mailbox_quota_exceeded"
	SubUnroutableIPAddress  = "unroutable_ip_address"
	// do_not_mail
	SubRoleBased         = "role_based"
	SubDisposable        = "disposable"
	SubRoleBasedCatchAll = "role_based_catch_all"
	SubMXForward         = "mx_forward"
	SubGlobalSuppression = "global_suppression"
	SubPossibleTrap      = "possible_trap"
	SubToxic             = "toxic"
	// unknown
	SubAntispamSystem       = "antispam_system"
	SubExceptionOccurred    = "exception_occurred"
	SubFailedSMTPConnection = "failed_smtp_connection"
	SubForcibleDisconnect   = "forcible_disconnect"
	SubGreylisted           = "greylisted"
	SubMailServerNoResponse = "mail_server_did_not_respond"
	SubMailServerTempError  = "mail_server_temporary_error"
	SubTimeoutExceeded      = "timeout_exceeded"
)
