from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.audio_plan_state import AudioPlanState
from typing import cast



def _get_kwargs(
    id: int,
    plan_id: int,

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/clips/{id}/audio-plans/{plan_id}/diagnostics".format(id=quote(str(id), safe=""),plan_id=quote(str(plan_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | AudioPlanState | None:
    if response.status_code == 200:
        response_200 = AudioPlanState.from_dict(response.json())



        return response_200

    if response.status_code == 400:
        response_400 = cast(Any, None)
        return response_400

    if response.status_code == 401:
        response_401 = cast(Any, None)
        return response_401

    if response.status_code == 403:
        response_403 = cast(Any, None)
        return response_403

    if response.status_code == 404:
        response_404 = cast(Any, None)
        return response_404

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | AudioPlanState]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    plan_id: int,
    *,
    client: AuthenticatedClient | Client,

) -> Response[Any | AudioPlanState]:
    """  Met en file un job audio.diagnostic.N distinct par piste retenue du plan, de façon idempotente.
    Aucun effet sur les transcriptions. Un Worker ne prend un job que s'il déclare audio.diagnostic.* et
    détient le détecteur épinglé installé explicitement.

    Args:
        id (int):
        plan_id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | AudioPlanState]
     """


    kwargs = _get_kwargs(
        id=id,
plan_id=plan_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    plan_id: int,
    *,
    client: AuthenticatedClient | Client,

) -> Any | AudioPlanState | None:
    """  Met en file un job audio.diagnostic.N distinct par piste retenue du plan, de façon idempotente.
    Aucun effet sur les transcriptions. Un Worker ne prend un job que s'il déclare audio.diagnostic.* et
    détient le détecteur épinglé installé explicitement.

    Args:
        id (int):
        plan_id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | AudioPlanState
     """


    return sync_detailed(
        id=id,
plan_id=plan_id,
client=client,

    ).parsed

async def asyncio_detailed(
    id: int,
    plan_id: int,
    *,
    client: AuthenticatedClient | Client,

) -> Response[Any | AudioPlanState]:
    """  Met en file un job audio.diagnostic.N distinct par piste retenue du plan, de façon idempotente.
    Aucun effet sur les transcriptions. Un Worker ne prend un job que s'il déclare audio.diagnostic.* et
    détient le détecteur épinglé installé explicitement.

    Args:
        id (int):
        plan_id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | AudioPlanState]
     """


    kwargs = _get_kwargs(
        id=id,
plan_id=plan_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    plan_id: int,
    *,
    client: AuthenticatedClient | Client,

) -> Any | AudioPlanState | None:
    """  Met en file un job audio.diagnostic.N distinct par piste retenue du plan, de façon idempotente.
    Aucun effet sur les transcriptions. Un Worker ne prend un job que s'il déclare audio.diagnostic.* et
    détient le détecteur épinglé installé explicitement.

    Args:
        id (int):
        plan_id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | AudioPlanState
     """


    return (await asyncio_detailed(
        id=id,
plan_id=plan_id,
client=client,

    )).parsed
