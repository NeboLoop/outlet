<script lang="ts">
	import ProfileForm from '../ProfileForm.svelte';
	import SegmentsEditor from '../SegmentsEditor.svelte';

	// Who outreach is for: what the business sells and to whom (the brand
	// profile), and its segments. An AI reading the website fills these in
	// over MCP; this is where a person reads and corrects them.
</script>

<div class="space-y-8">
	<section>
		<h2 class="text-lg font-semibold text-text mb-1">Brand profile</h2>
		<p class="form-hint mb-3">
			What the business sells, to whom, and its proof. Outreach is written from it: an AI reading your website fills it
			in, and you correct it here.
		</p>
		<ProfileForm />
	</section>

	<section>
		<h2 class="text-lg font-semibold text-text mb-1">Segments</h2>
		<p class="form-hint mb-3">
			Who a campaign targets. Prospects are added into a segment, a campaign runs on one, and results break down by it.
			Filters guide whoever finds the prospects; anyone outside them is noted when added, never refused.
		</p>
		<SegmentsEditor />
	</section>
</div>
