import pytest
import time
from empower1.blockchain import Blockchain, USER_PUBLIC_KEYS, VALIDATOR_WALLETS
from empower1.block import Block
from empower1.transaction import Transaction
from empower1.wallet import Wallet

# Test Blockchain initialization with crypto-signed Genesis
def test_blockchain_initialization_crypto(empty_blockchain_real_genesis):
    """Test Blockchain initialization: should have a signed genesis block."""
    bc = empty_blockchain_real_genesis
    assert len(bc.chain) == 1, "Blockchain should have one block (genesis) upon initialization."
    genesis_block = bc.chain[0]
    assert genesis_block.index == 0
    assert genesis_block.previous_hash == "0"
    assert genesis_block.validator_address is not None # Genesis validator address
    assert genesis_block.signature_hex is not None # Genesis block should be signed

    # Verify genesis block signature (its validator pubkey should be in USER_PUBLIC_KEYS)
    genesis_validator_pub_key = USER_PUBLIC_KEYS.get(genesis_block.validator_address)
    assert genesis_validator_pub_key is not None
    assert genesis_block.verify_block_signature(genesis_validator_pub_key) is True

    assert len(bc.pending_transactions) == 0
    assert isinstance(bc.validators_stake, dict)

def test_last_block_property_crypto(blockchain_with_one_validator, alice_wallet, bob_wallet):
    """Test the last_block property after mining."""
    bc = blockchain_with_one_validator # Has genesis and one registered validator
    validator_wallet = list(VALIDATOR_WALLETS.values())[0] # Get the registered validator wallet

    # Populate USER_PUBLIC_KEYS for alice and bob if not already there from other fixtures
    if alice_wallet.address not in USER_PUBLIC_KEYS: USER_PUBLIC_KEYS[alice_wallet.address] = alice_wallet.get_public_key_hex()
    if bob_wallet.address not in USER_PUBLIC_KEYS: USER_PUBLIC_KEYS[bob_wallet.address] = bob_wallet.get_public_key_hex()

    tx = Transaction(alice_wallet.address, bob_wallet.address, 1.0)
    tx.sign(alice_wallet)
    bc.add_transaction(tx, alice_wallet.get_public_key_hex())

    mined_block = bc.mine_pending_transactions(validator_wallet.address)
    assert mined_block is not None
    assert bc.last_block == mined_block
    assert bc.last_block.index == 1 # Genesis is 0, this is the first mined block


# Test adding transactions with signature verification
def test_add_transaction_crypto_valid(blockchain_with_one_validator, alice_wallet, bob_wallet):
    bc = blockchain_with_one_validator
    USER_PUBLIC_KEYS[alice_wallet.address] = alice_wallet.get_public_key_hex() # Ensure pubkey is known

    tx = Transaction(alice_wallet.address, bob_wallet.address, 10.0)
    tx.sign(alice_wallet)

    result = bc.add_transaction(tx, alice_wallet.get_public_key_hex())
    assert result is True
    assert tx in bc.pending_transactions

def test_add_transaction_crypto_invalid_signature(blockchain_with_one_validator, alice_wallet, bob_wallet, capsys):
    bc = blockchain_with_one_validator
    USER_PUBLIC_KEYS[alice_wallet.address] = alice_wallet.get_public_key_hex()
    USER_PUBLIC_KEYS[bob_wallet.address] = bob_wallet.get_public_key_hex() # For Bob's key

    tx = Transaction(alice_wallet.address, bob_wallet.address, 10.0)
    tx.sign(alice_wallet)

    # Try to add with Bob's public key (should fail verification)
    result = bc.add_transaction(tx, bob_wallet.get_public_key_hex())
    assert result is False
    assert tx not in bc.pending_transactions
    captured = capsys.readouterr()
    assert "Invalid signature for transaction" in captured.out

def test_add_transaction_crypto_unsigned(blockchain_with_one_validator, alice_wallet, bob_wallet, capsys):
    bc = blockchain_with_one_validator
    USER_PUBLIC_KEYS[alice_wallet.address] = alice_wallet.get_public_key_hex()

    tx = Transaction(alice_wallet.address, bob_wallet.address, 10.0) # Not signed

    result = bc.add_transaction(tx, alice_wallet.get_public_key_hex())
    assert result is False # verify_signature in Transaction returns False if no signature_hex
    assert tx not in bc.pending_transactions
    captured = capsys.readouterr()
    # The error message comes from Transaction.verify_signature (implicitly via Wallet.verify_signature)
    # or directly if signature_hex is None. Our Transaction.verify_signature prints nothing if no sig.
    # Blockchain.add_transaction prints the error.
    assert "Invalid signature for transaction" in captured.out


# Test mining pending transactions with block signing
def test_mine_pending_transactions_crypto(blockchain_with_transactions_pending):
    bc = blockchain_with_transactions_pending # Has pending tx and a registered validator

    # Get the validator wallet that was registered in the fixture
    # Assuming fixture `blockchain_with_one_validator` (used by `blockchain_with_transactions_pending`)
    # registered `validator_wallet` from conftest.
    # We need its address to call mine_pending_transactions.
    # This is a bit complex due to fixture dependencies. A simpler way:
    assert len(VALIDATOR_WALLETS) >= 1 # Check fixture setup
    validator_wallet_addr = list(VALIDATOR_WALLETS.keys())[0] # Get one of the known validator addresses
    validator_wallet_obj = VALIDATOR_WALLETS[validator_wallet_addr]

    initial_chain_length = len(bc.chain)
    num_pending_tx = len(bc.pending_transactions)
    assert num_pending_tx > 0

    mined_block = bc.mine_pending_transactions(validator_wallet_addr)

    assert mined_block is not None
    assert isinstance(mined_block, Block)
    assert len(bc.chain) == initial_chain_length + 1
    assert bc.last_block == mined_block
    assert len(mined_block.transactions) == num_pending_tx
    assert len(bc.pending_transactions) == 0
    assert mined_block.validator_address == validator_wallet_addr
    assert mined_block.signature_hex is not None # Block should be signed

    # Verify the mined block's signature
    validator_public_key_hex = USER_PUBLIC_KEYS[validator_wallet_addr]
    assert mined_block.verify_block_signature(validator_public_key_hex) is True

def test_mine_no_pending_transactions_crypto(blockchain_with_one_validator, capsys):
    bc = blockchain_with_one_validator
    validator_addr = list(VALIDATOR_WALLETS.keys())[0]

    assert len(bc.pending_transactions) == 0
    mined_block = bc.mine_pending_transactions(validator_addr)
    assert mined_block is None # As per current logic, requires transactions
    captured = capsys.readouterr()
    assert "No pending transactions to mine." in captured.out


# Test chain validation with all crypto checks
def test_is_chain_valid_crypto_initial(empty_blockchain_real_genesis):
    """Test that an initial chain (signed genesis block) is valid."""
    assert empty_blockchain_real_genesis.is_chain_valid() is True

def test_is_chain_valid_crypto_after_mining(blockchain_with_transactions_pending):
    bc = blockchain_with_transactions_pending # Has pending tx and a validator
    validator_addr = list(VALIDATOR_WALLETS.keys())[0] # Get the validator address

    # Mine the pending transactions
    bc.mine_pending_transactions(validator_addr)
    assert bc.is_chain_valid() is True # Should be valid after mining correctly

    # Add more tx and mine again with another (or same) validator
    # For simplicity, re-use validator. Need user wallets for new tx.
    alice_wallet = Wallet()
    bob_wallet = Wallet()
    USER_PUBLIC_KEYS[alice_wallet.address] = alice_wallet.get_public_key_hex()
    USER_PUBLIC_KEYS[bob_wallet.address] = bob_wallet.get_public_key_hex()

    tx3 = Transaction(alice_wallet.address, bob_wallet.address, 1.0)
    tx3.sign(alice_wallet)
    bc.add_transaction(tx3, alice_wallet.get_public_key_hex())

    bc.mine_pending_transactions(validator_addr)
    assert bc.is_chain_valid() is True


def test_is_chain_valid_crypto_tampered_tx_signature(blockchain_with_transactions_pending, capsys):
    bc = blockchain_with_transactions_pending
    validator_addr = list(VALIDATOR_WALLETS.keys())[0]
    bc.mine_pending_transactions(validator_addr) # Mine a block

    if len(bc.chain) > 1 and bc.chain[1].transactions:
        # Tamper a transaction's signature within the mined block
        bc.chain[1].transactions[0].signature_hex = "tamperedhexsignature" * 5
        assert bc.is_chain_valid() is False
        captured = capsys.readouterr()
        assert "Transaction" in captured.out and "invalid signature" in captured.out
    else:
        pytest.skip("Chain too short or no transactions in block to tamper.")

def test_is_chain_valid_crypto_tampered_block_signature(blockchain_with_transactions_pending, capsys):
    bc = blockchain_with_transactions_pending
    validator_addr = list(VALIDATOR_WALLETS.keys())[0]
    bc.mine_pending_transactions(validator_addr)

    if len(bc.chain) > 1:
        # Tamper the block's own signature
        bc.chain[1].signature_hex = "tamperedblocksignature" * 5
        assert bc.is_chain_valid() is False
        captured = capsys.readouterr()
        assert "Block" in captured.out and "invalid validator signature" in captured.out
    else:
        pytest.skip("Chain too short to tamper block signature.")

def test_is_chain_valid_crypto_tampered_block_content_hash_mismatch(blockchain_with_transactions_pending, capsys):
    bc = blockchain_with_transactions_pending
    validator_addr = list(VALIDATOR_WALLETS.keys())[0]
    bc.mine_pending_transactions(validator_addr)

    if len(bc.chain) > 1 and bc.chain[1].transactions:
        # Tamper content that affects block hash (e.g., a transaction's amount)
        # This should make current_block.hash != current_block._calculate_block_hash()
        bc.chain[1].transactions[0].amount += 1000
        assert bc.is_chain_valid() is False
        captured = capsys.readouterr()
        # When tx content is tampered, its signature becomes invalid. is_chain_valid checks this.
        assert "Transaction" in captured.out and "invalid signature" in captured.out
    else:
        pytest.skip("Chain too short or no txs to tamper for hash mismatch.")

# Test PoS methods (simplified versions) with Wallet objects
def test_register_validator_wallet_crypto(empty_blockchain_real_genesis, validator_wallet):
    bc = empty_blockchain_real_genesis
    addr = validator_wallet.address
    stake = 100.0

    bc.register_validator_wallet(validator_wallet, stake)
    assert addr in bc.validators_stake
    assert bc.validators_stake[addr] == stake
    assert addr in VALIDATOR_WALLETS and VALIDATOR_WALLETS[addr] == validator_wallet
    assert addr in USER_PUBLIC_KEYS and USER_PUBLIC_KEYS[addr] == validator_wallet.get_public_key_hex()

def test_select_validator_crypto(empty_blockchain_real_genesis, validator_wallet, alice_wallet):
    bc = empty_blockchain_real_genesis
    assert bc.select_validator() is None # No validators initially other than genesis's (if not staked for selection)

    # Register two validators with different stakes
    val1_wallet = validator_wallet
    val2_wallet = alice_wallet # Using alice_wallet as another validator for test

    bc.register_validator_wallet(val1_wallet, 100)
    bc.register_validator_wallet(val2_wallet, 900) # val2 should be selected more

    selections = [bc.select_validator() for _ in range(1000)]
    count_val1 = selections.count(val1_wallet.address)
    count_val2 = selections.count(val2_wallet.address)

    assert count_val1 + count_val2 == 1000
    assert count_val2 > count_val1 * 5 # Heuristic for stake weighting

# Test Blockchain repr (no specific crypto changes, but ensure it runs)
def test_blockchain_repr_crypto(blockchain_with_transactions_pending):
    bc = blockchain_with_transactions_pending
    # Mine transactions to have more content in repr
    validator_addr = list(VALIDATOR_WALLETS.keys())[0]
    bc.mine_pending_transactions(validator_addr)

    repr_str = repr(bc)
    assert "Blockchain State:" in repr_str
    assert "Total Blocks:" in repr_str
    assert "Pending Transactions:" in repr_str
    assert "Registered Validators (Stake):" in repr_str
    assert "Chain:" in repr_str
    assert "Block(Index: 0" in repr_str # Genesis block
    assert "Block(Index: 1" in repr_str # Mined block

# To run: pytest from the project root.
