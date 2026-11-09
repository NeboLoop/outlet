/**
 * Keeps the current tab of a sideways-scrolling tab bar in view. On a phone a
 * tab bar can be wider than the screen; this scrolls the bar (never the page)
 * so the tab marked aria-current="page" shows. Pass the current path so it
 * runs again after each navigation.
 */
export function scrollActiveTab(node: HTMLElement, _path: string) {
	function reveal() {
		const active = node.querySelector<HTMLElement>('[aria-current="page"]');
		if (!active || node.scrollWidth <= node.clientWidth) return;
		const left = active.offsetLeft - node.offsetLeft;
		if (left < node.scrollLeft || left + active.offsetWidth > node.scrollLeft + node.clientWidth) {
			node.scrollLeft = left - (node.clientWidth - active.offsetWidth) / 2;
		}
	}
	reveal();
	return { update: reveal };
}
