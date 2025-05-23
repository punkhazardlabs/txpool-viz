export async function fetchTransactions() {
  const res = await fetch("/api/transactions");
  if (!res.ok) throw new Error("Failed to fetch transactions");
  return await res.json();
}

export async function fetchTxDetails(txHash: string) {
  const res = await fetch(`/api/transaction/${txHash}`);
  if (!res.ok) throw new Error("Failed to fetch transaction details");
  return await res.json();
}
