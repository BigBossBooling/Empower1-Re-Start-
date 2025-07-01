import time
import json # For deterministic serialization of transactions list
import hashlib
from empower1.transaction import Transaction # Assuming Transaction class is defined
from empower1.wallet import Wallet # For type hinting if validator signs

class Block:
    """
    Represents a block in the EmPower1 Blockchain.
    A block contains a list of transactions and is signed by its validator.
    """
    def __init__(self, index: int, transactions: list[Transaction], timestamp: float,
                 previous_hash: str, validator_address: str, signature_hex: str = None):
        """
        Constructor for a Block.
        Args:
            index (int): The block's index in the chain.
            transactions (list[Transaction]): A list of Transaction objects included in the block.
            timestamp (float): The time the block was created.
            previous_hash (str): The hash of the preceding block.
            validator_address (str): The address of the validator who created/validated this block.
                                     This should be the validator's public key hex or wallet address.
            signature_hex (str, optional): Hex-encoded DER signature of the block's hash by the validator.
        """
        self.index = index
        self.transactions = transactions # List of Transaction objects
        self.timestamp = timestamp
        self.previous_hash = previous_hash
        self.validator_address = validator_address # Validator's public key (hex) or wallet address

        # Signature is for the block's hash, set after hash calculation and signing
        self.signature_hex = signature_hex

        # Calculate block hash. It depends on all above fields including transaction details.
        self.hash = self._calculate_block_hash()

    def _get_transaction_hashes_for_merkle_or_hash(self) -> str:
        """
        Helper to get a string representation of transaction IDs for block hashing.
        For a real Merkle root, this would be more complex.
        For now, concatenates transaction_ids (which are hashes of tx content).
        """
        if not self.transactions:
            return ""
        # Ensure transactions are Transaction objects and have a transaction_id
        return "".join(tx.transaction_id for tx in self.transactions if hasattr(tx, 'transaction_id'))

    def get_data_for_block_hashing(self) -> bytes:
        """
        Serializes essential block information (excluding the block's own hash and signature)
        into a deterministic byte string for calculating the block's hash.
        """
        # Using json.dumps for transactions list to ensure deterministic representation if it contains dicts,
        # but self.transactions should be a list of Transaction objects.
        # We'll use the concatenated transaction IDs for simplicity as a 'fingerprint' of transactions.
        transaction_fingerprint = self._get_transaction_hashes_for_merkle_or_hash()

        block_header_data = (
            f"{self.index}{self.timestamp:.6f}{self.previous_hash}"
            f"{self.validator_address}{transaction_fingerprint}"
        )
        return block_header_data.encode('utf-8')

    def _calculate_block_hash(self) -> str:
        """
        Calculates the SHA256 hash of the block's content (header + transaction fingerprint).
        """
        return hashlib.sha256(self.get_data_for_block_hashing()).hexdigest()

    # For block signing by validator:
    # The data to be signed by the validator is typically the block's own hash.
    def get_data_for_block_signing(self) -> bytes:
        """
        Returns the data that the validator should sign. This is the block's own hash.
        """
        if not self.hash: # Should not happen if constructor is called correctly
            raise ValueError("Block hash not calculated yet. Cannot get data for signing.")
        return self.hash.encode('utf-8') # Sign the hex representation of the hash as bytes

    def sign_block(self, validator_wallet: Wallet):
        """
        Signs the block's hash using the validator's wallet.
        Args:
            validator_wallet (Wallet): The wallet of the validator.
        Raises:
            ValueError: If validator's address doesn't match block's validator_address
                        or if block hash is not available.
        """
        # This check depends on whether validator_address is the wallet address or public key hex.
        # Let's assume validator_address stored in block is the wallet address.
        if validator_wallet.address != self.validator_address:
             raise ValueError("Validator wallet does not match block's validator_address.")
        if not self.hash:
            raise ValueError("Block hash must be calculated before signing.")

        data_to_sign_hash = hashlib.sha256(self.get_data_for_block_signing()).digest() # Hash the block's hash string
        signature_der = validator_wallet.sign_data(data_to_sign_hash)
        self.signature_hex = signature_der.hex()

    def verify_block_signature(self, validator_public_key_hex: str) -> bool:
        """
        Verifies the block's signature using the validator's public key.
        Args:
            validator_public_key_hex (str): The hex string of the validator's public key.
        Returns:
            bool: True if the signature is valid, False otherwise.
        """
        if not self.signature_hex or not self.hash:
            return False

        data_to_verify_hash = hashlib.sha256(self.get_data_for_block_signing()).digest() # Hash of the block's hash string
        try:
            signature_der = bytes.fromhex(self.signature_hex)
            return Wallet.verify_signature(
                public_key_bytes_hex=validator_public_key_hex,
                data_hash=data_to_verify_hash,
                signature_der=signature_der
            )
        except ValueError: # Hex decoding error
            return False
        except Exception:
            return False

    def __repr__(self):
        return (f"Block(Index: {self.index}, Transactions: {len(self.transactions)}, "
                f"Timestamp: {self.timestamp}, Hash: {self.hash[:10]}..., Prev_Hash: {self.previous_hash[:10] if self.previous_hash else 'None'}..., "
                f"Validator: {self.validator_address}, Signed: {'Yes' if self.signature_hex else 'No'})")

if __name__ == '__main__':
    # Create Wallets for Alice (user) and ValidatorX
    alice_wallet = Wallet()
    validator_x_wallet = Wallet()
    print(f"Alice's Wallet Address: {alice_wallet.address}")
    print(f"ValidatorX Wallet Address: {validator_x_wallet.address}, PubKeyHex: {validator_x_wallet.get_public_key_hex()[:10]}...")

    # Create some transactions (signed by Alice)
    tx1 = Transaction(
        sender_address=alice_wallet.address,
        receiver_address="Bob_Wallet_Addr",
        amount=10.0,
        metadata={"msg": "tx1 for block"}
    )
    tx1.sign(alice_wallet)

    tx2 = Transaction(
        sender_address=alice_wallet.address,
        receiver_address="Charlie_Wallet_Addr",
        amount=5.0,
        metadata={"msg": "tx2 for block"}
    )
    tx2.sign(alice_wallet)
    print(f"\nTransaction 1 ID: {tx1.transaction_id}")
    print(f"Transaction 2 ID: {tx2.transaction_id}")

    # Create a genesis block (usually simpler, without transactions and special validator)
    genesis_block_validator_wallet = Wallet() # A conceptual genesis validator
    genesis_block = Block(
        index=0,
        transactions=[],
        timestamp=time.time(),
        previous_hash="0", # Genesis block has no previous hash
        validator_address=genesis_block_validator_wallet.address # Genesis validator address
    )
    # Genesis block might be pre-signed or have a known signature/no signature
    # For this demo, let's sign it.
    genesis_block.sign_block(genesis_block_validator_wallet)
    print(f"\nGenesis Block: {genesis_block}")
    is_genesis_sig_valid = genesis_block.verify_block_signature(genesis_block_validator_wallet.get_public_key_hex())
    print(f"Is Genesis Block signature valid? {is_genesis_sig_valid}")


    # Create a second block, validated by ValidatorX
    block2_timestamp = time.time() + 60
    block2 = Block(
        index=1,
        transactions=[tx1, tx2], # List of signed Transaction objects
        timestamp=block2_timestamp,
        previous_hash=genesis_block.hash,
        validator_address=validator_x_wallet.address # ValidatorX's address
    )
    print(f"\nBlock 2 (unsigned): {block2}")
    print(f"Block 2 Hash: {block2.hash}")
    print(f"Block 2 Data for signing (block's hash): {block2.get_data_for_block_signing().decode()}")


    # ValidatorX signs Block 2
    block2.sign_block(validator_x_wallet)
    print(f"Block 2 (signed): {block2}")
    assert block2.signature_hex is not None

    # Verify Block 2's signature
    is_block2_sig_valid = block2.verify_block_signature(
        validator_public_key_hex=validator_x_wallet.get_public_key_hex()
    )
    print(f"Is Block 2 signature valid (using ValidatorX's pubkey)? {is_block2_sig_valid}")
    assert is_block2_sig_valid

    # Tamper with block2's transaction (after block hash calculation and signing)
    # This should NOT invalidate block2.signature_hex against block2.hash,
    # but it WOULD make block2.hash different if recalculated.
    # The Blockchain's is_chain_valid will catch this by re-calculating block hash.
    if block2.transactions:
        block2.transactions[0].amount = 999.0 # Tamper

    recalculated_hash_after_tamper = block2._calculate_block_hash()
    print(f"\nBlock 2 original hash: {block2.hash}")
    print(f"Block 2 recalculated hash after tampering tx: {recalculated_hash_after_tamper}")
    assert block2.hash != recalculated_hash_after_tamper

    # The signature is for the ORIGINAL block hash, so it should still be valid against that original hash.
    is_block2_sig_still_valid_for_original_hash = block2.verify_block_signature(
         validator_x_wallet.get_public_key_hex()
    )
    print(f"Is Block 2 signature still valid for its STORED hash (even after tx tamper)? {is_block2_sig_still_valid_for_original_hash}")
    assert is_block2_sig_still_valid_for_original_hash

    print("\nBlock class with ECDSA signing demo complete.")
