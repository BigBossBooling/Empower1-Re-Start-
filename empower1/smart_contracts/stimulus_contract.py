# Smart Contract for managing and distributing stimulus payments.
# This contract would interact with the IRE and potentially AI/ML optimized logic.

from empower1.smart_contracts.base_contract import BaseContract, SmartContractError
# from empower1.transaction import Transaction # If contract constructs transactions

# Address of the IRE system or a trusted AI oracle that provides eligibility data
IRE_ORACLE_ADDRESS = "Emp1_IRE_Oracle_Trusted001"
STIMULUS_TOKEN_ASSET_ID = "empower_coin_stimulus" # The asset used for stimulus

class StimulusContract(BaseContract):
    """
    A smart contract to manage the distribution of stimulus payments.
    It could be funded by the TaxContract or a central treasury.
    Eligibility criteria are determined by the IRE (via an oracle or direct calls).
    """
    def __init__(self, contract_address, owner_address, blockchain_interface=None, ire_oracle_address=IRE_ORACLE_ADDRESS):
        super().__init__(contract_address, owner_address, blockchain_interface)
        self.version = "0.1.0-stimulus"
        self.ire_oracle_address = ire_oracle_address
        self.stimulus_pool_balance = 0.0 # Balance of funds held by this contract for stimulus
        self.authorized_distributors = {owner_address} # Addresses that can trigger distributions

        # Mapping of user_address to last stimulus received timestamp to prevent abuse
        self.last_stimulus_timestamp = {}
        self.minimum_interval_seconds = 24 * 60 * 60 # E.g., once per day

        self._emit_event("StimulusContractDeployed", {"oracle": self.ire_oracle_address})
        print(f"StimulusContract initialized at {contract_address}, Oracle: {self.ire_oracle_address}")

    def fund_pool(self, caller_address: str, amount: float, asset_id: str):
        """
        Allows anyone to send funds to the stimulus pool.
        In a real system, this would be tied to a transaction that transfers assets to the contract's address.
        Args:
            caller_address (str): Address sending the funds.
            amount (float): Amount of asset to fund.
            asset_id (str): The asset ID being funded (should match STIMULUS_TOKEN_ASSET_ID or be convertible).
        """
        if amount <= 0:
            raise SmartContractError("Funding amount must be positive.")
        if asset_id != STIMULUS_TOKEN_ASSET_ID:
            # Or convert if a different asset is allowed and a DEX is available
            raise SmartContractError(f"Only {STIMULUS_TOKEN_ASSET_ID} can be used to fund this pool directly.")

        # Simulate receiving funds. In reality, blockchain validates actual transfer.
        # self._verify_actual_transfer(caller_address, self.contract_address, amount, asset_id)
        self.stimulus_pool_balance += amount
        self._emit_event("PoolFunded", {
            "funder": caller_address,
            "amount": amount,
            "asset_id": asset_id,
            "new_pool_balance": self.stimulus_pool_balance
        })
        print(f"Stimulus pool funded by {caller_address} with {amount} {asset_id}. New balance: {self.stimulus_pool_balance}")
        return {"status": "success", "new_pool_balance": self.stimulus_pool_balance}

    def add_authorized_distributor(self, caller_address: str, distributor_address: str):
        """Allows the owner to add new authorized distributors."""
        if caller_address != self.owner_address:
            raise SmartContractError("Only the owner can add distributors.")
        self.authorized_distributors.add(distributor_address)
        self._emit_event("DistributorAdded", {"distributor": distributor_address})
        return {"status": "success", "message": f"Distributor {distributor_address} added."}

    def distribute_stimulus_batch(self, caller_address: str, distribution_data: list):
        """
        Distributes stimulus to a batch of users based on data likely from the IRE oracle.
        Args:
            caller_address (str): Must be an authorized distributor or the IRE oracle.
            distribution_data (list): A list of dicts, each like:
                                      {'user_address': 'xyz', 'amount': 50.0, 'reason_code': 'LOW_W_A1'}
        """
        if caller_address not in self.authorized_distributors and caller_address != self.ire_oracle_address:
            raise SmartContractError(f"Caller {caller_address} is not authorized to distribute stimulus.")

        if not distribution_data:
            raise SmartContractError("Distribution data cannot be empty.")

        successful_distributions = []
        failed_distributions = []
        total_distributed_amount = 0

        current_time = self._get_block_timestamp()

        for item in distribution_data:
            user_addr = item.get("user_address")
            amount = item.get("amount")
            reason = item.get("reason_code", "N/A")

            if not user_addr or amount is None or amount <= 0:
                failed_distributions.append({"user_address": user_addr, "error": "Invalid data"})
                continue

            # Check distribution frequency
            last_time = self.last_stimulus_timestamp.get(user_addr, 0)
            if current_time - last_time < self.minimum_interval_seconds:
                failed_distributions.append({"user_address": user_addr, "error": "Stimulus distributed too recently"})
                continue

            if self.stimulus_pool_balance >= amount:
                self.stimulus_pool_balance -= amount
                total_distributed_amount += amount

                # Simulate the transfer. In a real chain, this updates ledgers.
                # This assumes the contract itself holds and can send STIMULUS_TOKEN_ASSET_ID.
                transfer_success = self._transfer_funds(
                    from_address=self.contract_address, # From contract's own holdings
                    to_address=user_addr,
                    amount=amount,
                    asset_id=STIMULUS_TOKEN_ASSET_ID
                )

                if transfer_success:
                    self.last_stimulus_timestamp[user_addr] = current_time
                    successful_distributions.append(item)
                    self._emit_event("StimulusDistributed", {
                        "recipient": user_addr, "amount": amount, "asset_id": STIMULUS_TOKEN_ASSET_ID,
                        "reason": reason, "distributor": caller_address
                    })
                else:
                    # Should not happen if balance check passed and _transfer_funds is robust
                    self.stimulus_pool_balance += amount # Revert pool deduction
                    failed_distributions.append({"user_address": user_addr, "error": "Transfer failed unexpectedly"})
            else:
                failed_distributions.append({"user_address": user_addr, "error": "Insufficient pool funds"})
                # Optionally break if pool is empty

        print(f"Stimulus distribution complete. Success: {len(successful_distributions)}, Failed: {len(failed_distributions)}")
        return {
            "status": "partial_success" if failed_distributions else "success",
            "total_distributed": total_distributed_amount,
            "successful_count": len(successful_distributions),
            "failed_count": len(failed_distributions),
            "new_pool_balance": self.stimulus_pool_balance,
            "failures": failed_distributions if failed_distributions else None
        }

    def get_pool_balance(self, caller_address: str = None):
        """Returns the current stimulus pool balance."""
        return {"pool_balance": self.stimulus_pool_balance, "asset_id": STIMULUS_TOKEN_ASSET_ID}

    def get_last_stimulus_time(self, caller_address: str, user_address: str):
        """Returns the timestamp of the last stimulus payment for a user."""
        return {"user_address": user_address, "last_stimulus_timestamp": self.last_stimulus_timestamp.get(user_address)}

    def update_oracle_address(self, caller_address: str, new_oracle_address: str):
        """Allows the owner to update the IRE oracle address."""
        if caller_address != self.owner_address:
            raise SmartContractError("Only the owner can update the oracle address.")
        self.ire_oracle_address = new_oracle_address
        self._emit_event("OracleAddressUpdated", {"new_oracle_address": new_oracle_address})
        return {"status": "success", "message": f"Oracle address updated to {new_oracle_address}."}

    def get_contract_info(self, caller_address: str = None):
        """Returns specific information for this contract."""
        base_info = super().get_contract_info(caller_address)
        base_info.update({
            "ire_oracle_address": self.ire_oracle_address,
            "stimulus_pool_balance": self.stimulus_pool_balance,
            "stimulus_asset_id": STIMULUS_TOKEN_ASSET_ID,
            "authorized_distributor_count": len(self.authorized_distributors)
        })
        return base_info


if __name__ == "__main__":
    # Example Usage
    class MockBlockchainStimulus(BaseContract.MockBlockchain if hasattr(BaseContract, 'MockBlockchain') else object):
        # Inherit or define methods needed by StimulusContract
        def get_balance(self, address, asset_id): return 10000.0 # Dummy
        def create_transfer_transaction(self, from_addr, to_addr, amt, asset): return True
        def log_event(self, contract_address, event_name, event_data):
             print(f"[StimulusContract Log] From {contract_address} - Event: {event_name}, Data: {event_data}")
        def get_current_timestamp(self):
            import time
            return time.time()


    mock_bc_stimulus = MockBlockchainStimulus()
    owner_addr = "Owner_Stimulus_Contract_Deployer"
    stimulus_contract_addr = "SC_StimulusDistributor_001"
    ire_oracle = IRE_ORACLE_ADDRESS # Default one

    stimulus_sc = StimulusContract(
        contract_address=stimulus_contract_addr,
        owner_address=owner_addr,
        blockchain_interface=mock_bc_stimulus,
        ire_oracle_address=ire_oracle
    )
    print(f"\n--- StimulusContract ({stimulus_sc.version}) Info ---")
    print(stimulus_sc.invoke_method("get_contract_info", owner_addr, {}))

    print("\n--- Funding Stimulus Pool ---")
    funder1 = "Generous_Alice_Funder"
    stimulus_sc.invoke_method("fund_pool", funder1, {"amount": 5000.0, "asset_id": STIMULUS_TOKEN_ASSET_ID})
    print(stimulus_sc.invoke_method("get_pool_balance", funder1, {}))

    print("\n--- Adding Authorized Distributor ---")
    distributor_sys = "System_Automated_Distributor"
    stimulus_sc.invoke_method("add_authorized_distributor", owner_addr, {"distributor_address": distributor_sys})

    print("\n--- Distributing Stimulus (by Owner) ---")
    user1, user2, user3 = "User_Eligible_A", "User_Eligible_B", "User_Ineligible_C"
    distribution_batch_1 = [
        {"user_address": user1, "amount": 75.0, "reason_code": "LOW_W_B1"},
        {"user_address": user2, "amount": 50.0, "reason_code": "LOW_W_C2"},
    ]
    result_batch_1 = stimulus_sc.invoke_method("distribute_stimulus_batch", owner_addr, {"distribution_data": distribution_batch_1})
    print(f"Batch 1 Result: {result_batch_1}")
    print(stimulus_sc.invoke_method("get_pool_balance", owner_addr, {}))

    print(f"Last stimulus for {user1}: {stimulus_sc.invoke_method('get_last_stimulus_time', owner_addr, {'user_address': user1})}")

    print("\n--- Attempting Re-distribution too soon (by System Distributor) ---")
    # import time; time.sleep(1) # Ensure timestamp changes if needed, though our mock is simple
    distribution_batch_2 = [{"user_address": user1, "amount": 20.0, "reason_code": "LOW_W_D3"}]
    result_batch_2 = stimulus_sc.invoke_method("distribute_stimulus_batch", distributor_sys, {"distribution_data": distribution_batch_2})
    print(f"Batch 2 Result: {result_batch_2}") # Should fail for user1 due to frequency
    print(stimulus_sc.invoke_method("get_pool_balance", owner_addr, {})) # Balance should be unchanged

    # Simulate time passing for the minimum interval
    stimulus_sc.minimum_interval_seconds = 0 # For testing, allow immediate re-distribution
    print("\n--- Distributing again after interval reset (by System Distributor) ---")
    result_batch_3 = stimulus_sc.invoke_method("distribute_stimulus_batch", distributor_sys, {"distribution_data": distribution_batch_2})
    print(f"Batch 3 Result: {result_batch_3}") # Should succeed now
    print(stimulus_sc.invoke_method("get_pool_balance", owner_addr, {}))

    print("\nStimulusContract placeholder demo complete.")
