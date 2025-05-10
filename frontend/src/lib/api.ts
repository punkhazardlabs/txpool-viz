export async function fetchTransactions() {
  const res = await fetch('http://localhost:42069/transactions');
  if (!res.ok) throw new Error('Failed to fetch transactions');
  return await res.json();
}
