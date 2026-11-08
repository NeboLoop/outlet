// Screen words the guides quote, kept in one place so a renamed button is
// one edit here, not a hunt through every guide. A guide writes
// {{label.name}} and the help renderer puts the value in its place.
//
// The mailbox service labels follow the mailbox service screen (the brand's
// service connections). Check them against that screen whenever it changes.
export const helpLabels: Record<string, string> = {
	// Where a brand's mailbox service connections are managed
	connectionsScreen: 'Settings, Sending',
	connectionsHeading: 'Mailbox services',
	addConnection: 'Add mailbox service',
	connectionService: 'Service',
	connectionApiKey: 'API key',
	saveConnection: 'Save',
	manualService: 'By hand (SMTP and IMAP logins)',

	// Per connection
	connectDomainSection: 'Connect a domain you own',
	connectDomain: 'Connect',
	buySection: 'Buy',
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
