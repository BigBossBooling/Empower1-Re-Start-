# Base class for EmPower1 smart contracts.
# This provides common structure and utilities for other contracts.

class SmartContractError(Exception):
    """Custom exception for smart contract execution errors."""
    pass

class BaseContract:
    """
    A base class for smart contracts on the EmPower1 Blockchain.
    Contracts would typically be deployed to an address and have their state
    managed by the blockchain. This simulation will be simpler.
    """
    def __init__(self, contract_address, owner_address, blockchain_interface=None):
        """
        Initialize the base contract.
        Args:
            contract_address (str): The address where this contract is "deployed".
            owner_address (str): The address of the account that deployed/owns the contract.
            blockchain_interface (object, optional): An interface to interact with the blockchain
                                                     (e.g., to read state, get block info, emit events).
        """
        self.contract_address = contract_address
        self.owner_address = owner_address
        self.blockchain = blockchain_interface # For interacting with blockchain state
        self.is_active = True
        self.version = "1.0.0" # Default contract version

        print(f"BaseContract initialized at {contract_address} by {owner_address}.")

    def invoke_method(self, method_name: str, caller_address: str, args: dict):
        """
        Simulates invoking a method on the smart contract.
        Checks if the method exists and calls it with context (caller_address).
        Args:
            method_name (str): The name of the method to call.
            caller_address (str): The address of the account calling this method.
            args (dict): Arguments for the method.
        Returns:
            The result of the method call.
        Raises:
            SmartContractError: If the method doesn't exist or if there's an execution error.
            AttributeError: If the method is not found.
        """
        if not self.is_active:
            raise SmartContractError(f"Contract {self.contract_address} is not active.")

        if not hasattr(self, method_name):
            raise AttributeError(f"Method '{method_name}' not found in contract {self.__class__.__name__}.")

        method_to_call = getattr(self, method_name)

        # Basic check: some methods might be owner-only
        # More sophisticated access control (e.g., roles) could be implemented.
        # Convention: methods starting with '_' are internal. Methods for external calls don't.
        # Convention: methods named `only_owner_*` should check caller_address.

        print(f"Invoking {method_name} on {self.contract_address} by {caller_address} with args {args}")
        try:
            # Pass caller_address as context, useful for auth checks within methods
            return method_to_call(caller_address=caller_address, **args)
        except Exception as e:
            # Log the error or handle it
            print(f"Error during execution of {method_name}: {e}")
            raise SmartContractError(f"Execution error in {method_name}: {str(e)}")


    def deactivate_contract(self, caller_address: str):
        """
        Allows the owner to deactivate the contract.
        """
        if caller_address != self.owner_address:
            raise SmartContractError("Only the owner can deactivate the contract.")
        self.is_active = False
        print(f"Contract {self.contract_address} deactivated by owner {caller_address}.")
        return {"status": "success", "message": "Contract deactivated."}

    def get_contract_info(self, caller_address: str = None):
        """
        Returns basic information about the contract.
        """
        return {
            "contract_address": self.contract_address,
            "owner_address": self.owner_address,
            "is_active": self.is_active,
            "version": self.version,
            "contract_type": self.__class__.__name__
        }

    # --- Helper methods for blockchain interaction (placeholders) ---

    def _get_block_timestamp(self):
        """Placeholder: Get current block timestamp from blockchain."""
        if self.blockchain and hasattr(self.blockchain, 'get_current_timestamp'):
            return self.blockchain.get_current_timestamp()
        import time
        return time.time() # Fallback for simulation

    def _get_caller_balance(self, caller_address: str, asset_id: str):
        """Placeholder: Get balance of an asset for a caller from blockchain."""
        if self.blockchain and hasattr(self.blockchain, 'get_balance'):
            return self.blockchain.get_balance(caller_address, asset_id)
        # Simulate if no blockchain interface
        print(f"Warning: _get_caller_balance for {caller_address} (Asset: {asset_id}) is simulated.")
        return 1000.0 # Dummy balance for simulation

    def _transfer_funds(self, from_address: str, to_address: str, amount: float, asset_id: str):
        """Placeholder: Instruct blockchain to transfer funds."""
        if self.blockchain and hasattr(self.blockchain, 'create_transfer_transaction'):
            # This implies the contract itself can initiate transactions,
            # or it prepares a transaction object to be signed and submitted.
            # For on-chain logic, this might mean updating internal ledgers
            # that the blockchain runtime then makes effective.
            print(f"Contract instructing fund transfer: {amount} {asset_id} from {from_address} to {to_address}")
            # return self.blockchain.create_transfer_transaction(from_address, to_address, amount, asset_id)
            return True # Simulate success
        print(f"Simulating fund transfer: {amount} {asset_id} from {from_address} to {to_address}")
        return True # Simulate success

    def _emit_event(self, event_name: str, event_data: dict):
        """Placeholder: Emit an event to the blockchain log."""
        log_message = f"Event emitted: {event_name}, Data: {event_data}"
        if self.blockchain and hasattr(self.blockchain, 'log_event'):
            self.blockchain.log_event(self.contract_address, event_name, event_data)
        else:
            print(f"(Simulated Event) Contract {self.contract_address}: {log_message}")


if __name__ == "__main__":
    # Example Usage
    class MockBlockchain:
        def get_current_timestamp(self):
            import time
            return time.time()
        def log_event(self, contract_address, event_name, event_data):
            print(f"[Blockchain Log] From {contract_address} - Event: {event_name}, Data: {event_data}")

    mock_bc = MockBlockchain()
    owner = "Owner_Alice_Addr"
    contract_addr = "Contract_BaseDemo_Addr"

    base_contract = BaseContract(contract_address=contract_addr, owner_address=owner, blockchain_interface=mock_bc)

    print("\n--- Testing BaseContract ---")
    # Get info (no caller restriction for this method in base)
    info = base_contract.invoke_method("get_contract_info", caller_address="Random_Bob_Addr", args={})
    print(f"Contract Info: {info}")

    # Attempt to deactivate by non-owner
    try:
        base_contract.invoke_method("deactivate_contract", caller_address="Random_Bob_Addr", args={})
    except SmartContractError as e:
        print(f"Caught expected error: {e}")

    # Deactivate by owner
    result = base_contract.invoke_method("deactivate_contract", caller_address=owner, args={})
    print(f"Deactivation result: {result}")
    assert not base_contract.is_active

    # Attempt to call on deactivated contract
    try:
        base_contract.invoke_method("get_contract_info", caller_address=owner, args={})
    except SmartContractError as e:
        print(f"Caught expected error on deactivated contract: {e}")

    print("\nBaseContract demo complete.")
