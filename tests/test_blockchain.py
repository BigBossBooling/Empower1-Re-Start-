import pytest
import time
from unittest.mock import patch # Added
from empower1.blockchain import Blockchain, USER_PUBLIC_KEYS, VALIDATOR_WALLETS
from empower1.block import Block
from empower1.transaction import Transaction
from empower1.consensus.manager import ValidatorManager # Added
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
    assert isinstance(bc.validator_manager, ValidatorManager) # Check for ValidatorManager instance
    # The genesis validator is internal to Blockchain's _create_and_sign_genesis_block
    # and isn't registered in validator_manager in the same way user validators are.
    # So, validator_manager.validators might be empty or only contain user-registered ones.
    # Let's check based on how register_validator_wallet works.
    # For a fresh blockchain, no user validators are registered yet.
    assert len(bc.validator_manager.validators) == 0


def test_last_block_property_crypto(blockchain_with_one_validator, alice_wallet, bob_wallet, validator_wallet):
    """Test the last_block property after mining."""
    bc = blockchain_with_one_validator # Has genesis and validator_wallet registered

    USER_PUBLIC_KEYS[alice_wallet.address] = alice_wallet.get_public_key_hex()
    USER_PUBLIC_KEYS[bob_wallet.address] = bob_wallet.get_public_key_hex()

    tx = Transaction(alice_wallet.address, bob_wallet.address, 1.0)
    tx.sign(alice_wallet)
    bc.add_transaction(tx, alice_wallet.get_public_key_hex())

    # Get the Validator object for validator_wallet from the manager
    validator_obj = bc.validator_manager.get_validator(validator_wallet.address)
    assert validator_obj is not None, "Validator wallet from fixture was not found in manager"

    with patch.object(bc.validator_manager, 'select_next_validator', return_value=validator_obj):
        mined_block = bc.mine_pending_transactions()
        assert mined_block is not None
        assert bc.last_block == mined_block
        assert bc.last_block.index == 1


# Test adding transactions with signature verification
def test_add_transaction_crypto_valid(blockchain_with_one_validator, alice_wallet, bob_wallet):
    bc = blockchain_with_one_validator
    USER_PUBLIC_KEYS[alice_wallet.address] = alice_wallet.get_public_key_hex()

    tx = Transaction(alice_wallet.address, bob_wallet.address, 10.0)
    tx.sign(alice_wallet)

    result = bc.add_transaction(tx, alice_wallet.get_public_key_hex())
    assert result is True
    assert tx in bc.pending_transactions

def test_add_transaction_crypto_invalid_signature(blockchain_with_one_validator, alice_wallet, bob_wallet, capsys):
    bc = blockchain_with_one_validator
    USER_PUBLIC_KEYS[alice_wallet.address] = alice_wallet.get_public_key_hex()
    USER_PUBLIC_KEYS[bob_wallet.address] = bob_wallet.get_public_key_hex()

    tx = Transaction(alice_wallet.address, bob_wallet.address, 10.0)
    tx.sign(alice_wallet)

    result = bc.add_transaction(tx, bob_wallet.get_public_key_hex()) # Wrong public key
    assert result is False
    assert tx not in bc.pending_transactions
    captured = capsys.readouterr()
    assert "Invalid signature for transaction" in captured.out

def test_add_transaction_crypto_unsigned(blockchain_with_one_validator, alice_wallet, bob_wallet, capsys):
    bc = blockchain_with_one_validator
    USER_PUBLIC_KEYS[alice_wallet.address] = alice_wallet.get_public_key_hex()

    tx = Transaction(alice_wallet.address, bob_wallet.address, 10.0) # Not signed

    result = bc.add_transaction(tx, alice_wallet.get_public_key_hex())
    assert result is False
    assert tx not in bc.pending_transactions
    captured = capsys.readouterr()
    assert "Invalid signature for transaction" in captured.out


# Test mining pending transactions with block signing
def test_mine_pending_transactions_crypto(blockchain_with_transactions_pending):
    bc = blockchain_with_transactions_pending

    selected_validator_obj_from_manager = bc.validator_manager.select_next_validator_round_robin()
    assert selected_validator_obj_from_manager is not None, "Fixture did not set up an active validator"
    # selected_validator_wallet = VALIDATOR_WALLETS[selected_validator_obj_from_manager.wallet_address] # Not needed for call

    initial_chain_length = len(bc.chain)
    num_pending_tx = len(bc.pending_transactions)
    assert num_pending_tx > 0

    with patch.object(bc.validator_manager, 'select_next_validator', return_value=selected_validator_obj_from_manager):
        mined_block = bc.mine_pending_transactions()

    assert mined_block is not None
    assert isinstance(mined_block, Block)
    assert len(bc.chain) == initial_chain_length + 1
    assert bc.last_block == mined_block
    assert len(mined_block.transactions) == num_pending_tx
    assert len(bc.pending_transactions) == 0
    assert mined_block.validator_address == selected_validator_obj_from_manager.wallet_address
    assert mined_block.signature_hex is not None

    validator_public_key_hex = USER_PUBLIC_KEYS[selected_validator_obj_from_manager.wallet_address]
    assert mined_block.verify_block_signature(validator_public_key_hex) is True

def test_mine_no_pending_transactions_crypto(blockchain_with_one_validator, validator_wallet, capsys):
    bc = blockchain_with_one_validator

    assert len(bc.pending_transactions) == 0
    validator_obj = bc.validator_manager.get_validator(validator_wallet.address)
    assert validator_obj is not None
    with patch.object(bc.validator_manager, 'select_next_validator', return_value=validator_obj):
        mined_block = bc.mine_pending_transactions()
    assert mined_block is None
    captured = capsys.readouterr()
    assert "No pending transactions for validator" in captured.out


# Test chain validation with all crypto checks
def test_is_chain_valid_crypto_initial(empty_blockchain_real_genesis):
    assert empty_blockchain_real_genesis.is_chain_valid() is True

def test_is_chain_valid_crypto_after_mining(blockchain_with_transactions_pending, alice_wallet, bob_wallet):
    bc = blockchain_with_transactions_pending

    selected_validator_obj_from_manager = bc.validator_manager.select_next_validator_round_robin()
    assert selected_validator_obj_from_manager is not None

    with patch.object(bc.validator_manager, 'select_next_validator', return_value=selected_validator_obj_from_manager):
        bc.mine_pending_transactions()
    assert bc.is_chain_valid() is True

    USER_PUBLIC_KEYS[alice_wallet.address] = alice_wallet.get_public_key_hex()
    USER_PUBLIC_KEYS[bob_wallet.address] = bob_wallet.get_public_key_hex()

    tx3 = Transaction(alice_wallet.address, bob_wallet.address, 1.0)
    tx3.sign(alice_wallet)
    bc.add_transaction(tx3, alice_wallet.get_public_key_hex())

    another_selected_validator = bc.validator_manager.select_next_validator_round_robin()
    if not another_selected_validator:
        another_selected_validator = selected_validator_obj_from_manager

    with patch.object(bc.validator_manager, 'select_next_validator', return_value=another_selected_validator):
        bc.mine_pending_transactions()
    assert bc.is_chain_valid() is True


def test_is_chain_valid_crypto_tampered_tx_signature(blockchain_with_transactions_pending, capsys):
    bc = blockchain_with_transactions_pending
    selected_validator_obj = bc.validator_manager.select_next_validator_round_robin()
    assert selected_validator_obj is not None
    with patch.object(bc.validator_manager, 'select_next_validator', return_value=selected_validator_obj):
        bc.mine_pending_transactions()

    if len(bc.chain) > 1 and bc.chain[1].transactions:
        bc.chain[1].transactions[0].signature_hex = "tamperedhexsignature" * 5
        assert bc.is_chain_valid() is False
        captured = capsys.readouterr()
        assert "Transaction" in captured.out and "invalid signature" in captured.out
    else:
        pytest.skip("Chain too short or no transactions in block to tamper.")

def test_is_chain_valid_crypto_tampered_block_signature(blockchain_with_transactions_pending, capsys):
    bc = blockchain_with_transactions_pending
    selected_validator_obj = bc.validator_manager.select_next_validator_round_robin()
    assert selected_validator_obj is not None
    with patch.object(bc.validator_manager, 'select_next_validator', return_value=selected_validator_obj):
        bc.mine_pending_transactions()

    if len(bc.chain) > 1:
        bc.chain[1].signature_hex = "tamperedblocksignature" * 5
        assert bc.is_chain_valid() is False
        captured = capsys.readouterr()
        assert "Block" in captured.out and "invalid validator signature" in captured.out
    else:
        pytest.skip("Chain too short to tamper block signature.")

def test_is_chain_valid_crypto_tampered_block_content_hash_mismatch(blockchain_with_transactions_pending, capsys):
    bc = blockchain_with_transactions_pending
    selected_validator_obj = bc.validator_manager.select_next_validator_round_robin()
    assert selected_validator_obj is not None
    with patch.object(bc.validator_manager, 'select_next_validator', return_value=selected_validator_obj):
        bc.mine_pending_transactions()

    if len(bc.chain) > 1 and bc.chain[1].transactions:
        bc.chain[1].transactions[0].amount += 1000
        assert bc.is_chain_valid() is False
        captured = capsys.readouterr()
        assert "Transaction" in captured.out and "invalid signature" in captured.out
    else:
        pytest.skip("Chain too short or no txs to tamper for hash mismatch.")

# Test PoS methods (simplified versions) with Wallet objects
def test_register_validator_wallet_crypto(empty_blockchain_real_genesis, validator_wallet):
    bc = empty_blockchain_real_genesis
    addr = validator_wallet.address
    stake = 100.0

    bc.register_validator_wallet(validator_wallet, stake)

    managed_validator = bc.validator_manager.get_validator(addr)
    assert managed_validator is not None
    assert managed_validator.wallet_address == addr
    assert managed_validator.stake == stake
    assert managed_validator.is_active == (stake >= bc.validator_manager.min_stake_active)

    assert addr in VALIDATOR_WALLETS and VALIDATOR_WALLETS[addr] == validator_wallet
    assert addr in USER_PUBLIC_KEYS and USER_PUBLIC_KEYS[addr] == validator_wallet.get_public_key_hex()


# Test Blockchain repr (no specific crypto changes, but ensure it runs)
def test_blockchain_repr_crypto(blockchain_with_transactions_pending):
    bc = blockchain_with_transactions_pending
    selected_validator_obj = bc.validator_manager.select_next_validator_round_robin()
    assert selected_validator_obj is not None
    with patch.object(bc.validator_manager, 'select_next_validator', return_value=selected_validator_obj):
        bc.mine_pending_transactions()

    repr_str = repr(bc)
    assert "Blockchain State:" in repr_str
    assert "Total Blocks:" in repr_str
    assert "Pending Transactions:" in repr_str
    assert "Validators (from Manager):" in repr_str # Updated string
    assert "Chain:" in repr_str
    assert "Block(Index: 0" in repr_str
    assert "Block(Index: 1" in repr_str

# To run: pytest from the project root.
