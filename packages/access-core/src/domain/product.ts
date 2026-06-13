export const productKeys = ['planner_link', 'finance_link', 'infra_link', 'loka_link'] as const;

export type ProductKey = (typeof productKeys)[number];

export function isProductKey(value: string): value is ProductKey {
	return productKeys.includes(value as ProductKey);
}
