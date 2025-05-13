export async function fetchTransactions() {
  const res = await fetch("http://localhost:42069/transactions");
  if (!res.ok) throw new Error("Failed to fetch transactions");
  return await res.json();
}

export async function fetchTxDetails(txHash: string) {
  const res = await fetch(`http://localhost:42069/transaction/${txHash}`);
  if (!res.ok) throw new Error("Failed to fetch transactions");
  return await res.json();
}
