from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.on_demand_engine_plan import OnDemandEnginePlan
from ...models.on_demand_engine_plan_input import OnDemandEnginePlanInput
from typing import cast



def _get_kwargs(
    *,
    body: OnDemandEnginePlanInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/voice-engines/on-demand-plan",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | OnDemandEnginePlan:
    if response.status_code == 200:
        response_200 = OnDemandEnginePlan.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | OnDemandEnginePlan]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: OnDemandEnginePlanInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | OnDemandEnginePlan]:
    """  Administrateur : plan d'un ordre d'engine à la demande (famille, version par défaut si omise) sur
    des Workers : disque cumulé, durée estimée et refus de matériel par Worker. Lecture seule : ne crée
    ni ordre ni identité de registre. Confirmation : createOnDemandEngineOrders.

    Args:
        origin (str):
        x_csrf_token (str):
        body (OnDemandEnginePlanInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | OnDemandEnginePlan]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    body: OnDemandEnginePlanInput,
    origin: str,
    x_csrf_token: str,

) -> Error | OnDemandEnginePlan | None:
    """  Administrateur : plan d'un ordre d'engine à la demande (famille, version par défaut si omise) sur
    des Workers : disque cumulé, durée estimée et refus de matériel par Worker. Lecture seule : ne crée
    ni ordre ni identité de registre. Confirmation : createOnDemandEngineOrders.

    Args:
        origin (str):
        x_csrf_token (str):
        body (OnDemandEnginePlanInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | OnDemandEnginePlan
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: OnDemandEnginePlanInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | OnDemandEnginePlan]:
    """  Administrateur : plan d'un ordre d'engine à la demande (famille, version par défaut si omise) sur
    des Workers : disque cumulé, durée estimée et refus de matériel par Worker. Lecture seule : ne crée
    ni ordre ni identité de registre. Confirmation : createOnDemandEngineOrders.

    Args:
        origin (str):
        x_csrf_token (str):
        body (OnDemandEnginePlanInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | OnDemandEnginePlan]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: OnDemandEnginePlanInput,
    origin: str,
    x_csrf_token: str,

) -> Error | OnDemandEnginePlan | None:
    """  Administrateur : plan d'un ordre d'engine à la demande (famille, version par défaut si omise) sur
    des Workers : disque cumulé, durée estimée et refus de matériel par Worker. Lecture seule : ne crée
    ni ordre ni identité de registre. Confirmation : createOnDemandEngineOrders.

    Args:
        origin (str):
        x_csrf_token (str):
        body (OnDemandEnginePlanInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | OnDemandEnginePlan
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
