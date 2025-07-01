import pytest
import time
import hashlib
from empower1.block import Block
from empower1.transaction import Transaction # For creating transaction instances for blocks
from empower1.wallet import Wallet # For validator signing

# Test basic Block creation and attributes with crypto
def test_block_creation_crypto(sample_transaction_signed, another_sample_transaction_signed, validator_wallet):
    """Test Block creation with transactions and validator address."""
    # sample_transaction_signed and another_sample_transaction_signed are from conftest
    # validator_wallet is also from conftest

    block = Block(
        index=1,
        transactions=[sample_transaction_signed, another_sample_transaction_signed],
        timestamp=time.time(),
        previous_hash="some_previous_hash_string", # Example previous hash
        validator_address=validator_wallet.address # Validator's wallet address
        # Signature is initially None, set by sign_block
    )
    assert block.index == 1
    assert len(block.transactions) == 2
    assert block.transactions[0] == sample_transaction_signed
    assert block.transactions[1] == another_sample_transaction_signed
    assert block.timestamp is not None
    assert block.previous_hash == "some_previous_hash_string"
    assert block.validator_address == validator_wallet.address
    assert block.signature_hex is None # Not signed yet
    assert block.hash is not None # Hash calculated on init

    # Test hash calculation components
    tx_fingerprint = "".join(tx.transaction_id for tx in block.transactions)
    expected_data_for_hashing = (
        f"{block.index}{block.timestamp:.6f}{block.previous_hash}"
        f"{block.validator_address}{tx_fingerprint}"
    ).encode('utf-8')
    expected_block_hash = hashlib.sha256(expected_data_for_hashing).hexdigest()
    assert block.hash == expected_block_hash


def test_block_hashing_determinism(validator_wallet):
    """Test that block hash is deterministic for same content."""
    ts = time.time()
    tx1 = Transaction(validator_wallet.address, "receiver1", 1.0, timestamp=ts-10)
    tx1.sign(validator_wallet) # Sign with some wallet

    block1_data = {
        "index": 1, "transactions": [tx1], "timestamp": ts,
        "previous_hash": "prev_hash_A", "validator_address": validator_wallet.address
    }
    block1 = Block(**block1_data)

    # Create another block with identical content (transactions must be identical objects or have identical IDs)
    # For simplicity, re-use tx1. If tx objects were complex, ensure deep copy or consistent creation.
    block2_data = {
        "index": 1, "transactions": [tx1], "timestamp": ts, # Identical tx list
        "previous_hash": "prev_hash_A", "validator_address": validator_wallet.address
    }
    block2 = Block(**block2_data)
    assert block1.hash == block2.hash

    # Change one thing and hash should differ
    block3_data = {
        "index": 1, "transactions": [tx1], "timestamp": ts + 1.0, # Different timestamp
        "previous_hash": "prev_hash_A", "validator_address": validator_wallet.address
    }
    block3 = Block(**block3_data)
    assert block1.hash != block3.hash


# Test block signing and verification
def test_block_signing_and_verification(validator_wallet, sample_transaction_signed):
    """Test signing a block and verifying its signature."""
    block = Block(
        index=1,
        transactions=[sample_transaction_signed],
        timestamp=time.time(),
        previous_hash="prev_hash_for_signing_test",
        validator_address=validator_wallet.address
    )
    assert block.signature_hex is None

    # Sign the block
    block.sign_block(validator_wallet)
    assert block.signature_hex is not None

    # Verify with the correct validator's public key
    validator_public_key_hex = validator_wallet.get_public_key_hex()
    assert block.verify_block_signature(validator_public_key_hex=validator_public_key_hex) is True

def test_block_verification_fail_wrong_key(validator_wallet, alice_wallet, sample_transaction_signed):
    """Test block signature verification fails with the wrong public key."""
    block = Block(
        index=1, transactions=[sample_transaction_signed], timestamp=time.time(),
        previous_hash="prev_hash_wrong_key", validator_address=validator_wallet.address
    )
    block.sign_block(validator_wallet) # Signed by validator_wallet

    # Try to verify with Alice's public key (alice_wallet is a different wallet)
    alice_public_key_hex = alice_wallet.get_public_key_hex()
    assert block.verify_block_signature(validator_public_key_hex=alice_public_key_hex) is False

def test_block_verification_fail_tampered_data_after_signing(validator_wallet, sample_transaction_signed):
    """
    Test verification fails if block data (used in block hash) is tampered after signing.
    The signature itself is for the original block hash. So, if the block's stored hash
    is compared, signature is valid. But if chain validation recalculates hash, it will differ.
    This test focuses on verify_block_signature which uses the block's *stored* hash.
    """
    block = Block(
        index=1, transactions=[sample_transaction_signed], timestamp=time.time(),
        previous_hash="prev_hash_tamper", validator_address=validator_wallet.address
    )
    original_block_hash = block.hash # Store original hash
    block.sign_block(validator_wallet) # Signs based on original_block_hash

    # Tamper with block content that affects its hash calculation
    block.timestamp = time.time() + 1000
    # block.hash would now be different if recalculated, but it's still the original_block_hash.
    # The signature was for original_block_hash.
    # verify_block_signature signs the block's hash, which is `self.hash` (original).
    # So, this test should still pass for verify_block_signature.
    # The chain validation (is_chain_valid) is where overall block integrity (recalculated hash) is checked.

    validator_public_key_hex = validator_wallet.get_public_key_hex()
    assert block.verify_block_signature(validator_public_key_hex=validator_public_key_hex) is True

    # If we were to MANUALLY change self.hash to something else, then verify_block_signature would fail:
    block.hash = "completely_fake_hash_after_signing"
    assert block.verify_block_signature(validator_public_key_hex=validator_public_key_hex) is False


def test_block_verification_fail_no_signature(validator_wallet, sample_transaction_signed):
    """Test block signature verification fails if the block is not signed."""
    block = Block(
        index=1, transactions=[sample_transaction_signed], timestamp=time.time(),
        previous_hash="prev_hash_no_sig", validator_address=validator_wallet.address
    )
    assert block.signature_hex is None
    validator_public_key_hex = validator_wallet.get_public_key_hex()
    assert block.verify_block_signature(validator_public_key_hex=validator_public_key_hex) is False


def test_get_data_for_block_signing(sample_block_signed): # sample_block_signed is already signed
    """Test the get_data_for_block_signing method."""
    # This method should return the block's own hash, encoded.
    expected_data = sample_block_signed.hash.encode('utf-8')
    assert sample_block_signed.get_data_for_block_signing() == expected_data

def test_block_repr_crypto(sample_block_signed): # sample_block_signed is already signed
    """Test the __repr__ method of a signed Block."""
    block_repr = repr(sample_block_signed)
    assert str(sample_block_signed.index) in block_repr
    assert str(len(sample_block_signed.transactions)) in block_repr
    assert sample_block_signed.hash[:10] in block_repr # Checks for short hash
    assert sample_block_signed.previous_hash[:10] in block_repr
    assert sample_block_signed.validator_address in block_repr
    assert "Signed: Yes" in block_repr

    # Test unsigned block repr
    unsigned_block = Block(0, [], time.time(), "0", "validator_addr_unsigned")
    unsigned_repr = repr(unsigned_block)
    assert "Signed: No" in unsigned_repr


# To run these tests: `pytest` in the terminal from the project root.
