import pytest
import time
from empower1.consensus.manager import ValidatorManager, MIN_STAKE_TO_BE_ACTIVE
from empower1.consensus.validator import Validator

# Sample validator data
VALIDATOR_1_ADDR = "Emp1_ValAddr_001_ManagerTest"
VALIDATOR_1_PK = "04_pk1_" + "a" * 122
VALIDATOR_2_ADDR = "Emp1_ValAddr_002_ManagerTest"
VALIDATOR_2_PK = "04_pk2_" + "b" * 122
VALIDATOR_3_ADDR = "Emp1_ValAddr_003_ManagerTest"
VALIDATOR_3_PK = "04_pk3_" + "c" * 122

@pytest.fixture
def manager():
    """Returns a new ValidatorManager instance for each test."""
    return ValidatorManager(min_stake_active=100.0) # Use a known min_stake for tests

def test_manager_initialization(manager):
    assert isinstance(manager.validators, dict)
    assert len(manager.validators) == 0
    assert manager.min_stake_active == 100.0
    assert len(manager._active_validator_addresses_round_robin) == 0

def test_add_new_validator_sufficient_stake(manager):
    val = manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 150.0)
    assert val is not None
    assert val.wallet_address == VALIDATOR_1_ADDR
    assert val.public_key_hex == VALIDATOR_1_PK
    assert val.stake == 150.0
    assert val.is_active is True
    assert VALIDATOR_1_ADDR in manager.validators
    assert VALIDATOR_1_ADDR in manager._active_validator_addresses_round_robin

def test_add_new_validator_insufficient_stake(manager):
    val = manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 50.0)
    assert val is not None
    assert val.stake == 50.0
    assert val.is_active is False
    assert VALIDATOR_1_ADDR not in manager._active_validator_addresses_round_robin

def test_add_new_validator_negative_initial_stake(manager):
    val = manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, -50.0)
    assert val is None # Should fail to add
    assert VALIDATOR_1_ADDR not in manager.validators

def test_update_validator_stake_increase_to_active(manager):
    manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 50.0) # Initially inactive
    assert manager.validators[VALIDATOR_1_ADDR].is_active is False

    val = manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 75.0) # 50 + 75 = 125 (active)
    assert val.stake == 125.0
    assert val.is_active is True
    assert VALIDATOR_1_ADDR in manager._active_validator_addresses_round_robin

def test_update_validator_stake_decrease_to_inactive(manager):
    manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 150.0) # Initially active
    assert manager.validators[VALIDATOR_1_ADDR].is_active is True

    val = manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, -100.0) # 150 - 100 = 50 (inactive)
    assert val.stake == 50.0
    assert val.is_active is False
    assert VALIDATOR_1_ADDR not in manager._active_validator_addresses_round_robin

def test_update_validator_stake_cannot_go_negative(manager):
    manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 20.0)
    val = manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, -50.0) # Try to reduce below 0
    assert val is None # Should fail
    assert manager.validators[VALIDATOR_1_ADDR].stake == 20.0 # Stake should remain unchanged

def test_get_validator(manager):
    manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 100.0)
    retrieved_val = manager.get_validator(VALIDATOR_1_ADDR)
    assert retrieved_val is not None
    assert retrieved_val.wallet_address == VALIDATOR_1_ADDR
    assert manager.get_validator("NON_EXISTENT_ADDR") is None

def test_get_active_validators(manager):
    manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 150.0) # Active
    manager.add_or_update_validator_stake(VALIDATOR_2_ADDR, VALIDATOR_2_PK, 50.0)  # Inactive
    manager.add_or_update_validator_stake(VALIDATOR_3_ADDR, VALIDATOR_3_PK, 200.0) # Active

    active_validators = manager.get_active_validators()
    assert len(active_validators) == 2
    active_addrs = [v.wallet_address for v in active_validators]
    assert VALIDATOR_1_ADDR in active_addrs
    assert VALIDATOR_3_ADDR in active_addrs
    assert VALIDATOR_2_ADDR not in active_addrs

def test_select_next_validator_round_robin_no_active(manager):
    assert manager.select_next_validator_round_robin() is None

def test_select_next_validator_round_robin_cycling(manager):
    manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 100.0)
    manager.add_or_update_validator_stake(VALIDATOR_2_ADDR, VALIDATOR_2_PK, 100.0)
    # Active list should be sorted by address: [VALIDATOR_1_ADDR, VALIDATOR_2_ADDR] if sort order is as expected
    # Forcing a known order for test predictability
    manager._active_validator_addresses_round_robin = sorted([VALIDATOR_1_ADDR, VALIDATOR_2_ADDR])
    manager._last_selected_validator_index_rr = -1


    selected1 = manager.select_next_validator_round_robin()
    assert selected1.wallet_address == manager._active_validator_addresses_round_robin[0]

    selected2 = manager.select_next_validator_round_robin()
    assert selected2.wallet_address == manager._active_validator_addresses_round_robin[1]

    selected3 = manager.select_next_validator_round_robin() # Cycle back
    assert selected3.wallet_address == manager._active_validator_addresses_round_robin[0]

    # Check last_block_produced_timestamp updated
    assert selected1.last_block_produced_timestamp > 0
    original_ts1 = selected1.last_block_produced_timestamp
    time.sleep(0.01)
    selected1_again = manager.select_next_validator_round_robin() # V2
    selected1_again = manager.select_next_validator_round_robin() # V1 again
    assert selected1_again.wallet_address == selected1.wallet_address
    assert selected1_again.last_block_produced_timestamp > original_ts1


def test_select_next_validator_round_robin_skips_inactive(manager):
    manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 100.0) # Active
    manager.add_or_update_validator_stake(VALIDATOR_2_ADDR, VALIDATOR_2_PK, 50.0)  # Inactive
    manager.add_or_update_validator_stake(VALIDATOR_3_ADDR, VALIDATOR_3_PK, 100.0) # Active

    # Active list should be [VALIDATOR_1_ADDR, VALIDATOR_3_ADDR] (sorted)
    manager._last_selected_validator_index_rr = -1


    for i in range(4): # Should cycle between V1 and V3
        selected = manager.select_next_validator_round_robin()
        assert selected.is_active is True
        assert selected.wallet_address in [VALIDATOR_1_ADDR, VALIDATOR_3_ADDR]

def test_set_minimum_stake_updates_active_validators(manager):
    manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 150.0) # Active (min 100)
    manager.add_or_update_validator_stake(VALIDATOR_2_ADDR, VALIDATOR_2_PK, 80.0)  # Inactive (min 100)

    assert manager.validators[VALIDATOR_1_ADDR].is_active is True
    assert manager.validators[VALIDATOR_2_ADDR].is_active is False
    assert len(manager.get_active_validators()) == 1

    manager.set_minimum_stake(50.0) # New minimum
    assert manager.validators[VALIDATOR_1_ADDR].is_active is True  # Still active
    assert manager.validators[VALIDATOR_2_ADDR].is_active is True  # Now active
    assert len(manager.get_active_validators()) == 2
    assert len(manager._active_validator_addresses_round_robin) == 2

    manager.set_minimum_stake(200.0)
    assert manager.validators[VALIDATOR_1_ADDR].is_active is False # Now inactive
    assert manager.validators[VALIDATOR_2_ADDR].is_active is False # Still inactive
    assert len(manager.get_active_validators()) == 0

def test_validator_address_pk_consistency_check(manager, capsys):
    manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, VALIDATOR_1_PK, 100.0)
    # Try to update with same address but different PK
    manager.add_or_update_validator_stake(VALIDATOR_1_ADDR, "04_DIFFERENT_PK" + "d"*116, 50.0)
    captured = capsys.readouterr()
    assert "Warning: Public key for existing validator" in captured.out
    assert manager.validators[VALIDATOR_1_ADDR].public_key_hex == VALIDATOR_1_PK # Should keep original PK
    assert manager.validators[VALIDATOR_1_ADDR].stake == 150.0 # Stake should still update
