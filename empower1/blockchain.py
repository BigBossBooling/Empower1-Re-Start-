import time
from typing import Dict
from empower1.block import Block
from empower1.transaction import Transaction
from empower1.wallet import Wallet

USER_PUBLIC_KEYS = {}
VALIDATOR_WALLETS = {}

from empower1.consensus.manager import ValidatorManager

class Blockchain:
    NATIVE_CURRENCY_SYMBOL = "EPC" # Define native currency symbol

    def __init__(self, network_interface=None):
        self.chain = []
        self.pending_transactions = []
        self.validator_manager = ValidatorManager()
        self.network_interface = network_interface

        self.balances: Dict[str, float] = {}
        self.total_supply_epc: float = 0.0

        self._create_and_sign_genesis_block()

    def _create_and_sign_genesis_block(self):
        genesis_validator_wallet = Wallet()
        VALIDATOR_WALLETS[genesis_validator_wallet.address] = genesis_validator_wallet
        USER_PUBLIC_KEYS[genesis_validator_wallet.address] = genesis_validator_wallet.get_public_key_hex()

        genesis_block = Block(
            index=0,
            transactions=[],
            timestamp=time.time(),
            previous_hash="0",
            validator_address=genesis_validator_wallet.address
        )
        genesis_block.sign_block(genesis_validator_wallet)
        self.chain.append(genesis_block)

        initial_supply_epc = 1_000_000.0
        genesis_validator_address = genesis_validator_wallet.address
        self.balances[genesis_validator_address] = initial_supply_epc
        self.total_supply_epc = initial_supply_epc
        print(f"Initial {initial_supply_epc} {self.NATIVE_CURRENCY_SYMBOL} allocated to genesis validator {genesis_validator_address}.")

    @property
    def last_block(self) -> Block:
        return self.chain[-1] if self.chain else None # Handle empty chain case for last_block

    def _process_transaction_for_state_changes(self, transaction: Transaction) -> bool:
        """
        Processes a single transaction to update balances for the native currency.
        This method assumes the transaction's signature and basic format are already validated.
        It only checks for sufficient funds and applies balance changes.
        Args:
            transaction (Transaction): The transaction to process.
        Returns:
            bool: True if the transaction was processed successfully (sufficient funds, EPC type),
                  False otherwise.
        """
        if transaction.asset_id != self.NATIVE_CURRENCY_SYMBOL:
            return True # Not an EPC transfer, so considered "processed" without state change for balances.

        sender_address = transaction.sender_address
        receiver_address = transaction.receiver_address
        amount = transaction.amount # Already float from Transaction init

        sender_balance = self.balances.get(sender_address, 0.0)

        # Validate funds (important: this check uses the current state of self.balances)
        if sender_balance < amount:
            print(f"Transaction {transaction.transaction_id} state change failed: Sender {sender_address} has insufficient balance ({sender_balance} {self.NATIVE_CURRENCY_SYMBOL}) for amount {amount} {self.NATIVE_CURRENCY_SYMBOL}.")
            return False

        self.balances[sender_address] = sender_balance - amount
        self.balances[receiver_address] = self.balances.get(receiver_address, 0.0) + amount

        # print(f"Processed EPC transfer: {amount} from {sender_address} to {receiver_address}. New balances: Sender={self.balances[sender_address]}, Receiver={self.balances[receiver_address]}")
        return True

    def add_transaction(self, transaction: Transaction, sender_public_key_hex: str, received_from_network: bool = False) -> bool:
        if not isinstance(transaction, Transaction):
            return False
        if not all(hasattr(transaction, attr) for attr in ['sender_address', 'receiver_address', 'amount']):
            return False

        if not transaction.verify_signature(sender_public_key_hex):
            print(f"Error: Invalid signature for transaction {transaction.transaction_id} from {transaction.sender_address}.")
            return False

        # Pre-check for sufficient funds (against current confirmed balances) before adding to pending pool
        if transaction.asset_id == self.NATIVE_CURRENCY_SYMBOL:
            current_sender_balance = self.balances.get(transaction.sender_address, 0.0)
            # Consider pending outgoing transactions from this sender to prevent double spending from mempool
            pending_outgoing_amount = sum(
                tx.amount for tx in self.pending_transactions
                if tx.sender_address == transaction.sender_address and tx.asset_id == self.NATIVE_CURRENCY_SYMBOL
            )
            available_balance = current_sender_balance - pending_outgoing_amount

            if available_balance < transaction.amount:
                print(f"Tx {transaction.transaction_id} rejected from mempool: Sender {transaction.sender_address} has insufficient available balance (Confirmed: {current_sender_balance}, Pending Out: {pending_outgoing_amount}, Available: {available_balance}) for amount {transaction.amount}.")
                return False

        if any(tx.transaction_id == transaction.transaction_id for tx in self.pending_transactions):
            return True # Already known

        self.pending_transactions.append(transaction)

        if self.network_interface and not received_from_network:
            self.network_interface.broadcast_transaction(transaction)
        return True

    def mine_pending_transactions(self) -> Block | None:
        selected_validator_obj = self.validator_manager.select_next_validator()
        if not selected_validator_obj:
            print("Error: No active validator selected to mine the block.")
            return None

        selected_validator_address = selected_validator_obj.wallet_address
        validator_wallet = VALIDATOR_WALLETS.get(selected_validator_address)

        if not validator_wallet:
            print(f"Error: Wallet for selected validator {selected_validator_address} not found.")
            return None

        if not self.pending_transactions:
            print(f"No pending transactions for validator {selected_validator_address} to mine.")
            return None

        # Create a temporary balance snapshot to validate transactions for this block
        # This snapshot starts from the current confirmed balances.
        temp_balances_for_block = self.balances.copy()
        valid_txs_for_block = []

        for tx in list(self.pending_transactions): # Iterate over a copy
            if tx.asset_id == self.NATIVE_CURRENCY_SYMBOL:
                sender_bal_snapshot = temp_balances_for_block.get(tx.sender_address, 0.0)
                if sender_bal_snapshot >= tx.amount:
                    temp_balances_for_block[tx.sender_address] = sender_bal_snapshot - tx.amount
                    temp_balances_for_block[tx.receiver_address] = temp_balances_for_block.get(tx.receiver_address, 0.0) + tx.amount
                    valid_txs_for_block.append(tx)
                else:
                    print(f"Tx {tx.transaction_id} by {tx.sender_address} for {tx.amount} EPC invalid due to insufficient funds ({sender_bal_snapshot} EPC) during mining. Removing from current block proposal.")
                    # Optionally, remove from self.pending_transactions permanently if invalid even against confirmed state
                    # For now, just exclude from this block.
            else: # Non-EPC transactions are included if they passed initial add_transaction checks
                valid_txs_for_block.append(tx)

        if not valid_txs_for_block: # If all pending transactions were invalid for state changes
            print(f"No valid transactions to include in block for validator {selected_validator_address}.")
            return None

        new_block = Block(
            index=len(self.chain),
            transactions=valid_txs_for_block,
            timestamp=time.time(),
            previous_hash=self.last_block.hash if self.last_block else "0",
            validator_address=selected_validator_address
        )
        new_block.sign_block(validator_wallet)

        # --- Apply state changes from this new block to the actual self.balances ---
        for tx in new_block.transactions:
            # _process_transaction_for_state_changes updates self.balances
            if not self._process_transaction_for_state_changes(tx):
                # This should not happen if pre-validation with temp_balances was correct
                print(f"CRITICAL ERROR: Transaction {tx.transaction_id} was valid with temp_balances but failed with self.balances during mining.")
                # Potentially revert any partial changes if transactional application is needed, or halt.
                # For now, this indicates a logic flaw if reached.
                return None

        self.chain.append(new_block)
        print(f"Block #{new_block.index} mined by {selected_validator_address} with {len(new_block.transactions)} txs.")

        mined_tx_ids = {tx.transaction_id for tx in new_block.transactions}
        self.pending_transactions = [ptx for ptx in self.pending_transactions if ptx.transaction_id not in mined_tx_ids]

        if self.network_interface:
            self.network_interface.broadcast_block(new_block)
        return new_block

    def is_chain_valid(self) -> bool:
        temp_balances_for_validation = {}
        # Properly initialize temp_balances with the actual genesis allocation
        if self.chain:
            genesis_val_addr_on_chain = self.chain[0].validator_address
            # Find the initial allocation from self.balances that corresponds to genesis
            # This assumes self.balances was correctly initialized.
            # A more robust way would be to define genesis allocation constants.
            initial_genesis_balance = 0
            # Search for the balance that was set in _create_and_sign_genesis_block
            # This is a bit indirect. Ideally, genesis allocation is fixed.
            # For now, assume total_supply_epc was given to genesis validator.
            if self.chain[0].transactions == []: # Typical genesis
                 temp_balances_for_validation[genesis_val_addr_on_chain] = self.total_supply_epc


        for i in range(len(self.chain)):
            current_block = self.chain[i]
            if current_block.hash != current_block._calculate_block_hash():
                print(f"Error: Block {current_block.index} has an invalid hash.")
                return False

            if i > 0:
                previous_block = self.chain[i-1]
                if current_block.previous_hash != previous_block.hash:
                    print(f"Error: Block {current_block.index} previous_hash mismatch.")
                    return False

                validator_public_key_hex = USER_PUBLIC_KEYS.get(current_block.validator_address)
                if not validator_public_key_hex:
                    print(f"Error: Public key for validator {current_block.validator_address} (Block {current_block.index}) not found.")
                    return False

                validator_in_manager = self.validator_manager.get_validator(current_block.validator_address)
                if not validator_in_manager:
                    print(f"Error: Validator {current_block.validator_address} (Block {current_block.index}) not found in ValidatorManager.")
                    return False
                # Active status check for historical blocks is complex; current check is simplified.
                # if not validator_in_manager.is_active:
                #     print(f"Error: Validator {current_block.validator_address} (Block {current_block.index}) is not currently active.")
                #     return False

                if not current_block.verify_block_signature(validator_public_key_hex):
                    print(f"Error: Block {current_block.index} has an invalid validator signature.")
                    return False

            elif i == 0 and current_block.signature_hex:
                genesis_validator_public_key_hex = USER_PUBLIC_KEYS.get(current_block.validator_address)
                if not genesis_validator_public_key_hex:
                     print(f"Error: Public key for genesis validator {current_block.validator_address} not found.")
                     return False
                if not current_block.verify_block_signature(genesis_validator_public_key_hex):
                    print(f"Error: Genesis Block has an invalid validator signature.")
                    return False

            # Validate transactions and replay state changes on temp_balances
            # For genesis block (i=0), transactions (if any) would be special (e.g. initial allocations beyond validator)
            # Our current genesis has no txs, so _process_transaction_for_state_changes won't run for it here.
            for tx in current_block.transactions:
                sender_public_key_hex = USER_PUBLIC_KEYS.get(tx.sender_address)
                if not sender_public_key_hex:
                    print(f"Error: Public key for sender {tx.sender_address} (Tx {tx.transaction_id}, Block {current_block.index}) not found.")
                    return False
                if not tx.verify_signature(sender_public_key_hex):
                    print(f"Error: Transaction {tx.transaction_id} in block {current_block.index} has an invalid signature.")
                    return False

                # Replay EPC transactions on temp_balances for consistency check
                if tx.asset_id == self.NATIVE_CURRENCY_SYMBOL:
                    sender_bal = temp_balances_for_validation.get(tx.sender_address, 0.0)
                    if sender_bal < tx.amount:
                        print(f"Chain validation error: Tx {tx.transaction_id} in block {current_block.index} - insufficient funds for sender {tx.sender_address} during replay (Balance: {sender_bal}, Amount: {tx.amount}).")
                        return False
                    temp_balances_for_validation[tx.sender_address] = sender_bal - tx.amount
                    temp_balances_for_validation[tx.receiver_address] = temp_balances_for_validation.get(tx.receiver_address, 0.0) + tx.amount

        # Compare replayed balances with actual self.balances (if chain has more than genesis)
        # This is a strong check. For very long chains, this might be slow.
        # Only compare if there's more than just the genesis block, as genesis sets the initial state.
        if len(self.chain) > 1:
            # Normalize balances by removing zero-balance accounts from both dicts for fair comparison
            norm_replayed_balances = {k: v for k, v in temp_balances_for_validation.items() if v != 0}
            norm_actual_balances = {k: v for k, v in self.balances.items() if v != 0}
            if norm_replayed_balances != norm_actual_balances:
                print("Error: Chain validation failed due to balance mismatch after replaying all transactions.")
                print(f"Expected (current) balances: {norm_actual_balances}")
                print(f"Replayed balances from chain: {norm_replayed_balances}")
                # Find differences for debugging
                # for addr in set(norm_replayed_balances.keys()) | set(norm_actual_balances.keys()):
                #     if norm_replayed_balances.get(addr) != norm_actual_balances.get(addr):
                #         print(f"  Mismatch for {addr}: Replayed={norm_replayed_balances.get(addr)}, Actual={norm_actual_balances.get(addr)}")
                return False

        print("Blockchain is valid (cryptographically, structurally, and balance states consistent).")
        return True

    def register_validator_wallet(self, validator_wallet: Wallet, stake_amount: float):
        if not isinstance(validator_wallet, Wallet):
            print("Error: Invalid validator wallet object.")
            return
        if stake_amount <= 0:
            print("Initial stake amount must be positive for validator registration.")
            return

        addr = validator_wallet.address
        pub_key_hex = validator_wallet.get_public_key_hex()
        validator_obj = self.validator_manager.add_or_update_validator_stake(addr, pub_key_hex, stake_amount)

        if validator_obj:
            VALIDATOR_WALLETS[addr] = validator_wallet
            USER_PUBLIC_KEYS[addr] = pub_key_hex
        # else: # Error already printed by add_or_update_validator_stake

    def __repr__(self):
        chain_str = "Blockchain State:\n"
        chain_str += f"  Total Blocks: {len(self.chain)}\n"
        chain_str += f"  Pending Transactions: {len(self.pending_transactions)}\n"
        chain_str += "  Validators (from Manager):\n"
        if self.validator_manager and self.validator_manager.validators:
            for addr, val_obj in self.validator_manager.validators.items():
                chain_str += f"    - {addr[:15]}... Stake: {val_obj.stake}, Active: {val_obj.is_active}\n"
        else:
            chain_str += "    No validators managed.\n"
        chain_str += "Chain:\n"
        for block in self.chain:
            chain_str += f"  ┗━ {block}\n"
        return chain_str

if __name__ == '__main__':
    from unittest.mock import patch # For __main__ example to work without network

    bc = Blockchain()
    print(bc)
    print("\nBalances after genesis:", bc.balances)
    print("Total EPC supply:", bc.total_supply_epc)

    wallet1 = Wallet()
    wallet2 = Wallet()
    genesis_validator_addr = bc.chain[0].validator_address

    USER_PUBLIC_KEYS[wallet1.address] = wallet1.get_public_key_hex()
    USER_PUBLIC_KEYS[wallet2.address] = wallet2.get_public_key_hex()

    genesis_validator_wallet = VALIDATOR_WALLETS[genesis_validator_addr] # Get the actual wallet

    tx1 = Transaction(genesis_validator_addr, wallet1.address, 1000.0, asset_id=Blockchain.NATIVE_CURRENCY_SYMBOL)
    tx1.sign(genesis_validator_wallet)

    print(f"\nAttempting to add Tx1: {tx1.amount} {tx1.asset_id} from {tx1.sender_address[:10]} to {wallet1.address[:10]}")
    if bc.add_transaction(tx1, USER_PUBLIC_KEYS[genesis_validator_addr]):
        print("Tx1 added to pending pool.")
    else:
        print("Tx1 failed to add.")
    print("Balances before mining:", bc.balances) # Should be unchanged yet

    node_cli_wallet = wallet1
    bc.register_validator_wallet(node_cli_wallet, 200.0)
    print(f"Node wallet {node_cli_wallet.address[:10]} staked 200.0 EPC.")

    # Ensure the validator manager knows about the genesis validator if it might be selected
    # (though typically genesis validator doesn't participate beyond genesis)
    # For this test, we'll ensure node_cli_wallet (wallet1) is selected for mining.

    print("\nAttempting to mine block...")
    # Patch select_next_validator to ensure node_cli_wallet is chosen
    # Need to get the Validator object for node_cli_wallet from the manager
    validator_obj_for_node_cli = bc.validator_manager.get_validator(node_cli_wallet.address)
    if not validator_obj_for_node_cli:
        print(f"ERROR: Test setup issue, {node_cli_wallet.address} not in validator manager after staking.")
    else:
        with patch.object(bc.validator_manager, 'select_next_validator', return_value=validator_obj_for_node_cli):
            mined_block = bc.mine_pending_transactions()

        if mined_block:
            print(f"Mined block {mined_block.index} by {mined_block.validator_address[:10]}")
        else:
            print("Mining failed.")

    print("\nBalances after mining:", bc.balances)
    print(f"Chain valid: {bc.is_chain_valid()}")
    print(bc)

    # Test another transaction
    tx2 = Transaction(wallet1.address, wallet2.address, 50.0, asset_id=Blockchain.NATIVE_CURRENCY_SYMBOL)
    tx2.sign(wallet1)
    print(f"\nAttempting to add Tx2: {tx2.amount} {tx2.asset_id} from {wallet1.address[:10]} to {wallet2.address[:10]}")
    if bc.add_transaction(tx2, USER_PUBLIC_KEYS[wallet1.address]):
        print("Tx2 added to pending pool.")
    else:
        print("Tx2 failed to add.")

    print("\nAttempting to mine second block...")
    # Assume wallet1 (node_cli_wallet) is still the only staker or will be selected again by round-robin
    validator_obj_for_node_cli = bc.validator_manager.get_validator(node_cli_wallet.address)
    with patch.object(bc.validator_manager, 'select_next_validator', return_value=validator_obj_for_node_cli):
        mined_block_2 = bc.mine_pending_transactions()

    if mined_block_2:
        print(f"Mined block {mined_block_2.index} by {mined_block_2.validator_address[:10]}")
    else:
        print("Second mining failed.")

    print("\nBalances after second mining:", bc.balances)
    print(f"Chain valid: {bc.is_chain_valid()}")
    print(bc)

    expected_genesis_bal = initial_supply_epc - 1000.0
    expected_wallet1_bal = 1000.0 - 50.0
    expected_wallet2_bal = 50.0
    print(f"\nExpected Balances Check:")
    print(f"Genesis ({genesis_validator_addr[:10]}): Expected={expected_genesis_bal}, Actual={bc.balances.get(genesis_validator_addr)}")
    print(f"Wallet1 ({wallet1.address[:10]}): Expected={expected_wallet1_bal}, Actual={bc.balances.get(wallet1.address)}")
    print(f"Wallet2 ({wallet2.address[:10]}): Expected={expected_wallet2_bal}, Actual={bc.balances.get(wallet2.address)}")

    assert bc.balances.get(genesis_validator_addr) == expected_genesis_bal
    assert bc.balances.get(wallet1.address) == expected_wallet1_bal
    assert bc.balances.get(wallet2.address) == expected_wallet2_bal
