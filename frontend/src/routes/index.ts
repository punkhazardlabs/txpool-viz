import TransactionsView from "./TransactionsView.svelte";
import InclusionListView from "./InclusionListView.svelte";

export default {
  "/": TransactionsView,
  "/inclusion-lists": InclusionListView
};
