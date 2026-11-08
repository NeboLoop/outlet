// Screen words the guides quote, kept in one place so a renamed button is
// one edit here, not a hunt through every guide. A guide writes
// {{label.name}} and the help renderer puts the value in its place.
//
// These follow the mailbox pool's connections on Settings, Sending
// (MailboxConnections.svelte and MailboxConnection.svelte). Check them
// against that screen whenever it changes.
export const helpLabels: Record<string, string> = {
	// Where a brand's mailbox service connections are managed
	connectionsScreen: 'Settings, Sending',
	connectionsHeading: 'Connections',
	addConnection: 'Add connection',
	connectionService: 'Service',
	connectionName: 'Name (optional)',
	connectionApiKey: 'API key',
	addConnectionSave: 'Add',
	manualService: 'By hand (Google Workspace, Microsoft 365, any SMTP and IMAP)',
	manualServiceShort: 'By hand',

	// Per connection, under Manage
	manage: 'Manage',
	connectionLimit: 'Daily limit per mailbox',
	connectionWarmupStart: 'Warm-up first day',
	connectionWarmupStep: 'Warm-up added each day',
	connectionPoolDefault: "The pool's",
	saveConnection: 'Save',
	removeConnection: 'Remove',
	moveTo: 'Move to...',
	move: 'Move',
	connectDomainSection: 'Connect a domain you own',
	connectDomain: 'Connect',
	buySection: 'Buy at',
	buyKindMailboxes: 'Mailboxes on a domain',
	buyKindDomain: 'A new domain',
	buyKindConnect: 'Connect a domain I own (Infraforge)',
	buyPeople: 'One mailbox a line: First Last, or First Last, local',
	getQuote: 'Get a quote',
	buy: 'Buy',
	syncSection: 'Sync with',
	syncInventory: 'What (service name) has',
	syncCompare: 'Compare',
	syncImport: 'Import missing mailboxes',
	deleteSection: 'Delete at',
	deleteMailbox: 'Delete'
};

const placeholder = /\{\{label\.([A-Za-z]+)\}\}/g;

// fillLabels puts each {{label.name}} in a guide's Markdown in its place.
// An unknown name stays as written, so the guide test catches it.
export function fillLabels(markdown: string): string {
	return markdown.replace(placeholder, (whole, name: string) => helpLabels[name] ?? whole);
}
