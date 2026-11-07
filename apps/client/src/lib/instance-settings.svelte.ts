import { toast } from '@facile/muse';
import { ApiError, backend } from '$lib/backend';
import { t } from '$lib/i18n/index.svelte';

export type NookTestResult = { success: boolean; message?: string };

function message(error: unknown, fallback: string): string {
	return error instanceof Error && error.message ? error.message : fallback;
}

/** The instance settings the Nook and Advanced panels edit together and save in one request. */
export class InstanceSettings {
	instanceName = $state('');
	isAdmin = $state(false);
	trashRetention = $state('30');
	nookWebhookUrl = $state('');
	nookSecret = $state('');
	nookEnabled = $state(false);
	saving = $state(false);
	testing = $state(false);
	nookTestResult = $state<NookTestResult | null>(null);

	constructor(private readonly token: string) {}

	/** Loads the stored values. A 403 only means the person is not an instance admin. */
	async load() {
		try {
			const settings = await backend.getSettings(this.token);
			this.instanceName = settings.instance_name ?? '';
			this.nookWebhookUrl = settings.nook_webhook_url ?? '';
			this.nookSecret = settings.nook_webhook_secret ?? '';
			this.nookEnabled = settings.nook_enabled === 'true';
			this.trashRetention = settings.trash_retention_days ?? '30';
			this.isAdmin = true;
		} catch (e) {
			if (!(e instanceof ApiError && e.status === 403)) toast.danger(t('settings.loadFailed'));
		}
	}

	/** Saves every value. A secret that still starts with an asterisk is the masked one and stays unsent. */
	async save() {
		this.saving = true;
		try {
			const values: Record<string, string> = {
				instance_name: this.instanceName,
				nook_webhook_url: this.nookWebhookUrl,
				nook_enabled: String(this.nookEnabled),
				trash_retention_days: this.trashRetention.trim()
			};
			if (!this.nookSecret.startsWith('*')) values.nook_webhook_secret = this.nookSecret;
			await backend.updateSettings(this.token, values);
			toast.success(t('settings.saved'));
		} catch (e) {
			toast.danger(message(e, t('settings.saveFailed')));
		}
		this.saving = false;
	}

	/** Asks the API to reach the Nook webhook with the values on screen. */
	async testNook() {
		this.testing = true;
		this.nookTestResult = null;
		try {
			this.nookTestResult = await backend.testNook(this.token, {
				url: this.nookWebhookUrl,
				secret: this.nookSecret,
				enabled: this.nookEnabled
			});
		} catch (e) {
			this.nookTestResult = { success: false, message: message(e, t('settings.nook.failed')) };
		}
		this.testing = false;
	}
}
