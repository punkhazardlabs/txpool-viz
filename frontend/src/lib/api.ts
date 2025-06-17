export interface TxSummary {
  hash: string;
  from: string;
  gasUsed: number;
  priorityFee: number;
  nonce: number;
  type: string;
}

export interface ApiTxResponse {
  hash: string;
  clients: string[];
  common: {
    tx: Record<string, any>;
    metadata: Record<string, any>;
  };
  diff: {
    tx: Record<string, Record<string, any>>;
    metadata: Record<string, Record<string, any>>;
  };
}

// Fetch list of transaction summaries (from backend)
export async function fetchTransactions(): Promise<TxSummary[]> {
  // fetch only the hashes
  const res = await fetch("/api/transactions");
  if (!res.ok) throw new Error(`Failed to fetch transactions: ${res.status}`);
  const summaries: TxSummary[] = await res.json();

  return summaries;
}

// Fetch detailed diff/common for a tx
export async function fetchTxDetails(txHash: string): Promise<ApiTxResponse> {
  const res = await fetch(`/api/transaction/${txHash}`);
  if (!res.ok) throw new Error(`Failed to fetch tx details: ${res.status}`);
  return await res.json();
}
