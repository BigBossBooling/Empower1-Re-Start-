import time
from empower1.block import Block
from empower1.transaction import Transaction
from empower1.wallet import Wallet # Now needed for actual signing

# A simple way to map addresses to public keys for this stage.
# In a real system, this would be part of node state or discoverable.
USER_PUBLIC_KEYS = {} # {user_wallet_address: user_public_key_hex}
VALIDATOR_WALLETS = {} # {validator_wallet_address: Wallet_object} for block signing

class Blockchain:
    """
    Manages the chain of blocks, transactions, and consensus.
    Integrates cryptographic signing and verification.
    """
    def __init__(self, network_interface=None): # network_interface can be empower1.network.Network
        self.chain = []
        self.pending_transactions = []
        self.validators_stake = {} # {validator_wallet_address: stake_amount}
        self.network_interface = network_interface # Store network interface

        # Create and sign the genesis block
        self._create_and_sign_genesis_block()

    def _create_and_sign_genesis_block(self):
        """
        Creates the first block, the Genesis Block.
        It's signed by a conceptual "Genesis Validator".
        """
        genesis_validator_wallet = Wallet() # Create a wallet for the genesis validator
        # Store its details for potential verification if needed, though genesis is often trusted.
        VALIDATOR_WALLETS[genesis_validator_wallet.address] = genesis_validator_wallet
        USER_PUBLIC_KEYS[genesis_validator_wallet.address] = genesis_validator_wallet.get_public_key_hex()


        genesis_block = Block(
            index=0,
            transactions=[],
            timestamp=time.time(),
            previous_hash="0",
            validator_address=genesis_validator_wallet.address
        )
        genesis_block.sign_block(genesis_validator_wallet) # Sign the block
        self.chain.append(genesis_block)

    @property
    def last_block(self) -> Block:
        """Returns the last block in the chain."""
        return self.chain[-1]

    def add_transaction(self, transaction: Transaction, sender_public_key_hex: str, received_from_network: bool = False) -> bool:
        """
        Adds a new transaction to the list of pending transactions after verifying its signature.
        Args:
            transaction (Transaction): The transaction to add.
            sender_public_key_hex (str): The sender's public key hex for signature verification.
            received_from_network (bool): Flag to indicate if this tx was received from another node.
                                          If True, it won't be immediately re-broadcast by this function.
        Returns:
            bool: True if transaction is valid and added, False otherwise.
        """
        if not isinstance(transaction, Transaction):
            print("Error: Invalid transaction object provided.")
            return False
        if not all(hasattr(transaction, attr) for attr in ['sender_address', 'receiver_address', 'amount']):
            print("Error: Transaction object missing required fields.")
            return False

        # Verify transaction signature
        if not transaction.verify_signature(sender_public_key_hex):
            print(f"Error: Invalid signature for transaction {transaction.transaction_id} from {transaction.sender_address}.")
            return False

        # TODO: Add other validations (e.g., sufficient funds - requires state management)
        # TODO: Check if transaction already exists in pending_transactions or chain

        self.pending_transactions.append(transaction)
        # print(f"Transaction {transaction.transaction_id} added to pending pool by {self.network_interface.self_node.node_id if self.network_interface else 'local'}.")

        # Broadcast new valid transaction if network interface is available AND it wasn't received from network
        if self.network_interface and not received_from_network:
            # print(f"Blockchain instance about to call broadcast_transaction for {transaction.transaction_id} (origin local)")
            self.network_interface.broadcast_transaction(transaction)

        return True

    def mine_pending_transactions(self, validator_wallet_address: str) -> Block | None:
        """
        Mines pending transactions into a new block. The block is signed by the validator.
        Args:
            validator_wallet_address (str): The wallet address of the validator creating this block.
                                           The corresponding Wallet object must be in VALIDATOR_WALLETS.
        Returns:
            Block: The newly created and added block, or None if no transactions or invalid validator.
        """
        if validator_wallet_address not in VALIDATOR_WALLETS:
            print(f"Error: Validator wallet for address {validator_wallet_address} not found for signing.")
            return None

        validator_wallet = VALIDATOR_WALLETS[validator_wallet_address]

        if not self.pending_transactions:
            # Allow mining empty blocks if needed by consensus, but usually requires transactions.
            # print("No pending transactions to mine. (Consider if empty blocks are allowed)")
            # For now, let's require transactions to mine a block beyond genesis.
            # If we want to allow empty blocks:
            # pass
            print("No pending transactions to mine.")
            return None


        new_block = Block(
            index=len(self.chain),
            transactions=list(self.pending_transactions), # Copy
            timestamp=time.time(),
            previous_hash=self.last_block.hash,
            validator_address=validator_wallet.address # Store validator's wallet address
        )

        # Validator signs the block
        new_block.sign_block(validator_wallet)

        self.chain.append(new_block)
        print(f"Block #{new_block.index} mined by {validator_wallet_address} with {len(new_block.transactions)} txs.")
        self.pending_transactions = []

        # Broadcast the newly mined block if network interface is available
        if self.network_interface:
            # print(f"Blockchain instance about to call broadcast_block for {new_block.hash}")
            self.network_interface.broadcast_block(new_block)

        return new_block

    def is_chain_valid(self) -> bool:
        """
        Validates the integrity of the blockchain:
        - Checks block hash integrity.
        - Checks previous_hash links.
        - Verifies validator signature on each block.
        - Verifies all transaction signatures within each block.
        """
        for i in range(len(self.chain)):
            current_block = self.chain[i]

            # 1. Verify block's own hash (recalculate and compare)
            if current_block.hash != current_block._calculate_block_hash():
                print(f"Error: Block {current_block.index} has an invalid hash.")
                return False

            # 2. For blocks after genesis, check previous_hash link and validator signature
            if i > 0:
                previous_block = self.chain[i-1]
                if current_block.previous_hash != previous_block.hash:
                    print(f"Error: Block {current_block.index} previous_hash mismatch.")
                    return False

                # Verify block signature
                # Need the validator's public key. Assume validator_address in block can be mapped to it.
                validator_public_key_hex = USER_PUBLIC_KEYS.get(current_block.validator_address)
                if not validator_public_key_hex:
                    print(f"Error: Public key for validator {current_block.validator_address} not found for block {current_block.index}.")
                    return False
                if not current_block.verify_block_signature(validator_public_key_hex):
                    print(f"Error: Block {current_block.index} has an invalid validator signature.")
                    return False
            # For Genesis block, signature check (if signed)
            elif i == 0 and current_block.signature_hex: # If genesis block is signed
                genesis_validator_public_key_hex = USER_PUBLIC_KEYS.get(current_block.validator_address)
                if not genesis_validator_public_key_hex:
                     print(f"Error: Public key for genesis validator {current_block.validator_address} not found.")
                     return False
                if not current_block.verify_block_signature(genesis_validator_public_key_hex):
                    print(f"Error: Genesis Block has an invalid validator signature.")
                    return False


            # 3. Verify all transactions within the block
            for tx in current_block.transactions:
                # Need sender's public key. Assume tx.sender_address can be mapped to it.
                sender_public_key_hex = USER_PUBLIC_KEYS.get(tx.sender_address)
                if not sender_public_key_hex:
                    print(f"Error: Public key for sender {tx.sender_address} of tx {tx.transaction_id} in block {current_block.index} not found.")
                    return False
                if not tx.verify_signature(sender_public_key_hex):
                    print(f"Error: Transaction {tx.transaction_id} in block {current_block.index} has an invalid signature.")
                    return False

        print("Blockchain is valid.")
        return True

    def register_validator_wallet(self, validator_wallet: Wallet, stake_amount: float):
        """
        Registers a validator by their Wallet object and stake.
        Stores the wallet for signing and public key for verification.
        """
        if not isinstance(validator_wallet, Wallet):
            print("Error: Invalid validator wallet object.")
            return
        if stake_amount <= 0:
            print("Stake amount must be positive.")
            return

        addr = validator_wallet.address
        self.validators_stake[addr] = self.validators_stake.get(addr, 0) + stake_amount
        VALIDATOR_WALLETS[addr] = validator_wallet # Store wallet for signing blocks
        USER_PUBLIC_KEYS[addr] = validator_wallet.get_public_key_hex() # Store public key
        print(f"Validator {addr} registered/updated stake to {self.validators_stake[addr]}. Wallet and PubKey stored.")

    def select_validator(self) -> str | None:
        """
        Selects a validator based on stake (simplified).
        Returns the wallet address of the selected validator.
        """
        if not self.validators_stake:
            print("No validators registered to select from.")
            return None
        import random
        total_stake = sum(self.validators_stake.values())
        if total_stake == 0:
            return random.choice(list(self.validators_stake.keys()))

        pick = random.uniform(0, total_stake)
        current_stake_sum = 0
        for addr, stake in self.validators_stake.items():
            current_stake_sum += stake
            if current_stake_sum >= pick:
                return addr
        return list(self.validators_stake.keys())[-1] # Fallback

    def __repr__(self):
        chain_str = "Blockchain State:\n"
        chain_str += f"  Total Blocks: {len(self.chain)}\n"
        chain_str += f"  Pending Transactions: {len(self.pending_transactions)}\n"
        chain_str += "  Registered Validators (Stake):\n"
        for addr, stake in self.validators_stake.items():
            chain_str += f"    - {addr}: {stake}\n"
        chain_str += "Chain:\n"
        for block in self.chain:
            chain_str += f"  ┗━ {block}\n"
            # Optionally print transaction details within each block for full repr
            # for tx_idx, tx in enumerate(block.transactions):
            #     chain_str += f"    ┗━ TX {tx_idx}: {tx.transaction_id[:10]}... From: {tx.sender_address[:10]}... To: {tx.receiver_address[:10]}... Amount: {tx.amount}\n"
        return chain_str


if __name__ == '__main__':
    # Initialize blockchain (Genesis block is created and signed)
    empower_chain = Blockchain()
    print("EmPower1 Blockchain initialized.")
    print(f"Genesis Block: {empower_chain.last_block}")
    # Print USER_PUBLIC_KEYS to see genesis validator's details
    # print("Initial USER_PUBLIC_KEYS:", USER_PUBLIC_KEYS)
    # print("Initial VALIDATOR_WALLETS:", VALIDATOR_WALLETS)


    # Create Wallets for users and validators
    alice_wallet = Wallet()
    bob_wallet = Wallet()
    validator1_wallet = Wallet()
    validator2_wallet = Wallet()

    # Simulate users making their public keys known (or blockchain discovering them)
    USER_PUBLIC_KEYS[alice_wallet.address] = alice_wallet.get_public_key_hex()
    USER_PUBLIC_KEYS[bob_wallet.address] = bob_wallet.get_public_key_hex()
    # Validator public keys are stored during registration by register_validator_wallet

    print(f"\nAlice's Wallet: {alice_wallet.address}")
    print(f"Bob's Wallet: {bob_wallet.address}")
    print(f"Validator1 Wallet: {validator1_wallet.address}")
    print(f"Validator2 Wallet: {validator2_wallet.address}")

    # Register validators
    empower_chain.register_validator_wallet(validator1_wallet, 1000)
    empower_chain.register_validator_wallet(validator2_wallet, 1500)

    # Create, sign, and add transactions
    print("\n--- Creating Transactions ---")
    tx1 = Transaction(sender_address=alice_wallet.address, receiver_address=bob_wallet.address, amount=50.0, asset_id="EMP")
    tx1.sign(alice_wallet) # Alice signs her transaction
    empower_chain.add_transaction(tx1, alice_wallet.get_public_key_hex()) # Add with her public key for verification

    tx2 = Transaction(sender_address=bob_wallet.address, receiver_address=alice_wallet.address, amount=20.0, asset_id="EMP_TokenA")
    tx2.sign(bob_wallet) # Bob signs his transaction
    empower_chain.add_transaction(tx2, bob_wallet.get_public_key_hex())

    print(f"\nPending transactions: {len(empower_chain.pending_transactions)}")

    # A validator (selected by PoS mechanism) mines a block
    selected_validator_addr = empower_chain.select_validator()
    if selected_validator_addr:
        print(f"\nSelected validator by PoS: {selected_validator_addr}")
        mined_block1 = empower_chain.mine_pending_transactions(selected_validator_addr)
        if mined_block1:
            print(f"Mined Block 1: {mined_block1}")
    else:
        print("Could not select a validator.")

    print(f"Pending transactions after mining: {len(empower_chain.pending_transactions)}")

    # Add more transactions
    tx3 = Transaction(sender_address=alice_wallet.address, receiver_address=validator1_wallet.address, amount=10.0, asset_id="EMP", fee=0.01)
    tx3.sign(alice_wallet)
    empower_chain.add_transaction(tx3, alice_wallet.get_public_key_hex())
    print(f"\nPending transactions: {len(empower_chain.pending_transactions)}")

    # Another validator mines
    selected_validator_addr_2 = empower_chain.select_validator()
    if selected_validator_addr_2:
        print(f"\nSelected validator by PoS: {selected_validator_addr_2}")
        mined_block2 = empower_chain.mine_pending_transactions(selected_validator_addr_2)
        if mined_block2:
            print(f"Mined Block 2: {mined_block2}")
    else:
        print("Could not select a validator for the second block.")

    # Validate the chain
    print("\n--- Validating Full Blockchain ---")
    empower_chain.is_chain_valid()

    print(f"\n--- Current Blockchain State ---")
    print(empower_chain)

    # Tampering Example: Tamper a transaction signature in a block
    if len(empower_chain.chain) > 1 and empower_chain.chain[1].transactions:
        print("\n--- Tampering Test: Modifying a transaction signature in Block 1 ---")
        original_sig_tx0 = empower_chain.chain[1].transactions[0].signature_hex
        empower_chain.chain[1].transactions[0].signature_hex = "tamperedsignaturehex" + "00" * 20

        print("Validating tampered blockchain (expect failure on tx signature)...")
        empower_chain.is_chain_valid()

        # Restore for other potential tests (though not strictly necessary here)
        empower_chain.chain[1].transactions[0].signature_hex = original_sig_tx0

    # Tampering Example: Tamper a block signature
    if len(empower_chain.chain) > 1:
        print("\n--- Tampering Test: Modifying Block 1's validator signature ---")
        original_block_sig = empower_chain.chain[1].signature_hex
        empower_chain.chain[1].signature_hex = "tamperedblocksignaturehex" + "00" * 30

        print("Validating tampered blockchain (expect failure on block signature)...")
        empower_chain.is_chain_valid()
        empower_chain.chain[1].signature_hex = original_block_sig


    print("\n--- End of Enhanced Blockchain Demo ---")
