import { profileColumns } from './column-profile.js';

export interface ProfileRequest {
	id: number;
	columns: { name: string; type: string }[];
	rows: unknown[][];
}

self.onmessage = (event: MessageEvent<ProfileRequest>) => {
	const { id, columns, rows } = event.data;
	self.postMessage({ id, profiles: profileColumns(columns, rows) });
};
